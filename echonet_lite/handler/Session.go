package handler

import (
	"context"
	"crypto/rand"
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/network"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	mathrand "math/rand"
	"net"
	"sync"
	"time"
)

// SessionTimeoutType はセッションタイムアウトの種類を表す型
type SessionTimeoutType int

const (
	SessionTimeoutMaxRetries SessionTimeoutType = iota // 最大再送回数に達した
)

// SessionTimeoutEvent はセッションタイムアウトに関するイベントを表す構造体
type SessionTimeoutEvent struct {
	Device echonet_lite.IPAndEOJ // タイムアウトが発生したデバイス
	Type   SessionTimeoutType    // タイムアウトの種類
	Error  error                 // エラー情報
}

// ErrMaxRetriesReached は最大再送回数に達したことを示すエラー
type ErrMaxRetriesReached struct {
	MaxRetries    int
	Device        echonet_lite.IPAndEOJ
	TotalDuration time.Duration
	RetryInterval time.Duration
}

func (e ErrMaxRetriesReached) Error() string {
	return fmt.Sprintf("maximum retries reached (%d) for device %v after %v (retry interval: %v)",
		e.MaxRetries, e.Device, e.TotalDuration, e.RetryInterval)
}

// ジッタ計算用の定数
const (
	JitterPercentage   = 0.3              // ±30%のジッタ
	MinIntervalRatio   = 0.5              // 最小間隔は基準値の50%
	MaxDelayMultiplier = 5                // 同一IP遅延の最大倍率
	BackoffMultiplier  = 2.0              // Exponential backoffの倍率
	MaxRetryInterval   = 60 * time.Second // 最大リトライ間隔
)

// ブロードキャストアドレスの設定
var BroadcastIP = network.GetIPv4BroadcastIP()

type Key struct {
	TID echonet_lite.TIDType
}

func MakeKey(msg *echonet_lite.ECHONETLiteMessage) Key {
	return Key{msg.TID}
}

type CallbackCompleteStatus int // プロパティコールバック完了ステータス
const (
	CallbackFinished CallbackCompleteStatus = iota
	CallbackContinue
)

type CallbackFunc func(net.IP, *echonet_lite.ECHONETLiteMessage) (CallbackCompleteStatus, error)
type PersistentCallbackFunc func(net.IP, *echonet_lite.ECHONETLiteMessage) error

type Entry struct {
	ESVs     []echonet_lite.ESVType
	Callback CallbackFunc
}

type DispatchTable map[Key]Entry

func (dt DispatchTable) Register(key Key, ESVs []echonet_lite.ESVType, callback CallbackFunc) {
	dt[key] = Entry{ESVs, callback}
}

func (dt DispatchTable) Unregister(key Key) {
	delete(dt, key)
}

type Session struct {
	mu              sync.RWMutex
	dispatchTable   DispatchTable
	receiveCallback PersistentCallbackFunc
	infCallback     PersistentCallbackFunc
	tid             echonet_lite.TIDType
	eoj             echonet_lite.EOJ
	conn            *network.UDPConnection
	MulticastIP     net.IP
	Debug           bool
	ctx             context.Context                   // コンテキスト
	cancel          context.CancelFunc                // コンテキストのキャンセル関数
	MaxRetries      int                               // 最大再送回数
	RetryInterval   time.Duration                     // 再送間隔
	TimeoutCh       chan SessionTimeoutEvent          // タイムアウト通知用チャンネル
	failedEPCs      map[string][]echonet_lite.EPCType // 失敗したEPCsを保持するマップ
	IsOfflineFunc   func(echonet_lite.IPAndEOJ) bool  // デバイスがオフラインかどうかを判定する関数（オプショナル）
	rng             *mathrand.Rand                    // スレッドセーフな乱数生成器
}

// IsLocalIP は指定されたIPアドレスが自身のローカルIPのいずれかと一致するかを確認します
func (s *Session) IsLocalIP(ip net.IP) bool {
	return s.conn.IsLocalIP(ip)
}

// calculateRetryIntervalWithJitter は、再送間隔にジッタを加えた値を返します
// ジッタは基準値の±30%の範囲でランダムに決定されます
// retryCount: 0から始まるリトライ回数（0は初回のリトライ）
func (s *Session) calculateRetryIntervalWithJitter(retryCount int) time.Duration {
	// 入力検証: RetryIntervalが正の値であることを確認
	if s.RetryInterval <= 0 {
		slog.Warn("RetryIntervalが無効な値です。デフォルト値を使用", "interval", s.RetryInterval)
		return 3 * time.Second // デフォルト値
	}

	// Exponential backoffを適用: baseInterval * (BackoffMultiplier ^ retryCount)
	baseInterval := s.RetryInterval
	for i := 0; i < retryCount; i++ {
		baseInterval = time.Duration(float64(baseInterval) * BackoffMultiplier)
		// 最大値を超えないようにする
		baseInterval = min(baseInterval, MaxRetryInterval)
	}

	// スレッドセーフな乱数生成（crypto/randを使用）
	s.mu.Lock()
	defer s.mu.Unlock()

	// 初期化されていない場合は乱数生成器を作成
	if s.rng == nil {
		// crypto/randを使って安全なシードを生成
		seedBig, err := rand.Int(rand.Reader, big.NewInt(1<<62))
		if err != nil {
			// crypto/randが失敗した場合は時刻ベースのシード
			s.rng = mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
		} else {
			s.rng = mathrand.New(mathrand.NewSource(seedBig.Int64()))
		}
	}

	// ±30%のジッタを計算（定数を使用）
	jitterRange := float64(baseInterval) * JitterPercentage
	// -30% から +30% の範囲でランダムな値を生成
	jitter := (s.rng.Float64() - 0.5) * 2 * jitterRange

	// 基準値にジッタを加算
	intervalWithJitter := time.Duration(float64(baseInterval) + jitter)

	// 最小値を基準値の50%に設定（極端に短い間隔を防ぐ）
	minInterval := time.Duration(float64(baseInterval) * MinIntervalRatio)
	intervalWithJitter = max(intervalWithJitter, minInterval)

	return intervalWithJitter
}

// SetTimeoutChannel はタイムアウト通知用チャンネルを設定する
func (s *Session) SetTimeoutChannel(ch chan SessionTimeoutEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TimeoutCh = ch
}

func CreateSession(ctx context.Context, ip net.IP, EOJ echonet_lite.EOJ, debug bool, networkMonitorConfig *network.NetworkMonitorConfig, isOfflineFunc func(echonet_lite.IPAndEOJ) bool) (*Session, error) {
	// タイムアウトなしのコンテキストを作成（キャンセルのみ可能）
	sessionCtx, cancel := context.WithCancel(ctx)

	multicastIP := echonet_lite.ECHONETLiteMulticastIPv4

	conn, err := network.CreateUDPConnection(sessionCtx, ip, echonet_lite.ECHONETLitePort, multicastIP, networkMonitorConfig)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, err
	}
	return &Session{
		dispatchTable: make(DispatchTable),
		tid:           echonet_lite.TIDType(1),
		eoj:           EOJ,
		conn:          conn,
		MulticastIP:   multicastIP,
		Debug:         debug,
		ctx:           sessionCtx,
		cancel:        cancel,
		MaxRetries:    7,               // デフォルトの最大再送回数（指数バックオフで約2分のタイムアウト、応答の遅い冷蔵庫などに対応）
		RetryInterval: 3 * time.Second, // デフォルトの再送間隔
		failedEPCs:    make(map[string][]echonet_lite.EPCType),
		IsOfflineFunc: isOfflineFunc,
	}, nil
}

func (s *Session) OnInf(callback PersistentCallbackFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.infCallback = callback
}

func (s *Session) OnReceive(callback PersistentCallbackFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receiveCallback = callback
}

func (s *Session) MainLoop() {

	for {
		// DispatchTableがnilかどうかをロックして確認
		s.mu.RLock()
		dt := s.dispatchTable
		isNil := dt == nil
		s.mu.RUnlock()

		if isNil {
			break
		}

		// コンテキストがキャンセルされていないか確認
		select {
		case <-s.ctx.Done():
			return
		default:
			// 継続
		}

		// タイムアウトなしでコンテキストを作成（キャンセルのみ可能）
		receiveCtx, cancel := context.WithCancel(s.ctx)
		data, addr, err := s.conn.Receive(receiveCtx)
		cancel() // 必ずキャンセルする

		if err != nil {
			// コンテキストのキャンセルまたはタイムアウトの場合
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				// タイムアウトの場合は次の受信を待つ
				continue
			}

			// 接続が閉じられた場合
			if err.Error() == "use of closed network connection" {
				slog.Info("受信終了: 接続が閉じられました")
				break
			}

			// その他のネットワークエラー
			// net.Error.Temporary()はdeprecatedなので、特定のエラータイプで判断する
			if errors.Is(err, net.ErrClosed) {
				// 接続が閉じられた場合
				slog.Info("受信終了: 接続が閉じられました")
				break
			}

			// エラーログを記録
			slog.Error("データ受信中にエラーが発生", "err", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if len(data) == 0 {
			// データが空の場合は次の受信を待つ
			continue
		}

		if s.Debug {
			hexDump := hex.EncodeToString(data)
			slog.Debug("受信データ(hex)", "addr", addr, "hex", hexDump)
		}

		msg, err := echonet_lite.ParseECHONETLiteMessage(data)
		if err != nil {
			slog.Error("パケット解析エラー", "err", err)
			continue
		}

		if s.Debug {
			fmt.Printf("応答を受信: %s から --- %v\n", addr, msg)
		}

		switch msg.ESV {
		case echonet_lite.ESVSet_Res, echonet_lite.ESVSetI_SNA, echonet_lite.ESVSetC_SNA,
			echonet_lite.ESVGet_Res, echonet_lite.ESVGet_SNA,
			echonet_lite.ESVINFC_Res,
			echonet_lite.ESVINF_REQ_SNA,
			echonet_lite.ESVSetGet_Res, echonet_lite.ESVSetGet_SNA:
			// Get the callback while holding the lock
			s.mu.RLock()
			key := MakeKey(msg)
			entry, ok := s.dispatchTable[key]
			s.mu.RUnlock()

			// Execute callback outside the lock
			if ok {
				for _, esv := range entry.ESVs {
					if esv == msg.ESV {
						var complete CallbackCompleteStatus
						complete, err = entry.Callback(addr.IP, msg)
						if complete == CallbackFinished {
							s.UnregisterCallback(key)
						}
						break
					}
				}
			}
			if err != nil {
				slog.Error("ディスパッチエラー", "err", err)
			}
		case echonet_lite.ESVINF, echonet_lite.ESVINFC:
			// Get the callback while holding the lock
			s.mu.RLock()
			callback := s.infCallback
			s.mu.RUnlock()

			// Execute callback outside the lock
			if callback != nil {
				err = callback(addr.IP, msg)
				if err != nil {
					slog.Error("Infコールバックエラー", "err", err)
				}
			}
		case echonet_lite.ESVGet, echonet_lite.ESVSetC, echonet_lite.ESVSetI, echonet_lite.ESVINF_REQ:
			s.mu.RLock()
			callback := s.receiveCallback
			s.mu.RUnlock()
			if callback != nil {
				err = callback(addr.IP, msg)
				if err != nil {
					slog.Error("ReceiveCallbackエラー", "DEOJ", msg.DEOJ, "err", err)
				}
			}
		}
	}
}

func (s *Session) Close() error {
	s.mu.Lock()
	s.dispatchTable = nil // まずディスパッチテーブルをクリアして新しい処理を停止

	// コンテキストをキャンセル
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()

	// コネクションを閉じてエラーを返す
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	return nil
}

func (s *Session) newTID() echonet_lite.TIDType {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tid++
	return s.tid
}

func (s *Session) registerCallback(key Key, ESVs []echonet_lite.ESVType, callback CallbackFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dispatchTable.Register(key, ESVs, callback)
}

func (s *Session) sendMessage(ip net.IP, msg *echonet_lite.ECHONETLiteMessage) error {
	if _, err := s.conn.SendTo(ip, msg.Encode()); err != nil {
		slog.Error("パケット送信エラー", "err", err)
		return err
	}
	if s.Debug {
		fmt.Printf("パケットを送信: %v へ --- %v\n", ip, msg)
	}
	return nil
}

func (s *Session) SendResponse(ip net.IP, msg *echonet_lite.ECHONETLiteMessage, ESV echonet_lite.ESVType, property echonet_lite.Properties, setGetProperty echonet_lite.Properties) error {
	msgSend := &echonet_lite.ECHONETLiteMessage{
		TID:              msg.TID,
		SEOJ:             msg.DEOJ,
		DEOJ:             msg.SEOJ,
		ESV:              ESV,
		Properties:       property,
		SetGetProperties: setGetProperty,
	}
	return s.sendMessage(ip, msgSend)
}

func (s *Session) Broadcast(SEOJ echonet_lite.EOJ, ESV echonet_lite.ESVType, property echonet_lite.Properties) error {
	msg := &echonet_lite.ECHONETLiteMessage{
		TID:        s.newTID(),
		SEOJ:       SEOJ,
		DEOJ:       echonet_lite.NodeProfileObject,
		ESV:        ESV,
		Properties: property,
	}
	ip := s.MulticastIP
	if ip == nil {
		ip = BroadcastIP
	}
	return s.sendMessage(ip, msg)
}

// GetPropertiesCallbackFunc はプロパティ取得のコールバック関数の型。
type GetPropertiesCallbackFunc func(device echonet_lite.IPAndEOJ, success bool, properties echonet_lite.Properties, FailedEPCs []echonet_lite.EPCType) (CallbackCompleteStatus, error)

func (s *Session) CreateGetPropertyMessage(device echonet_lite.IPAndEOJ, EPCs []echonet_lite.EPCType) *echonet_lite.ECHONETLiteMessage {
	props := make([]echonet_lite.Property, 0, len(EPCs))
	for _, epc := range EPCs {
		props = append(props, echonet_lite.Property{EPC: epc})
	}
	return &echonet_lite.ECHONETLiteMessage{
		TID:        s.newTID(),
		SEOJ:       s.eoj,
		DEOJ:       device.EOJ,
		ESV:        echonet_lite.ESVGet,
		Properties: props,
	}
}

func (s *Session) prepareStartGetProperties(device echonet_lite.IPAndEOJ, EPCs []echonet_lite.EPCType, callback GetPropertiesCallbackFunc) (*echonet_lite.ECHONETLiteMessage, Key) {
	msg := s.CreateGetPropertyMessage(device, EPCs)
	key := MakeKey(msg)
	s.registerCallback(key, msg.ESV.ResponseESVs(), func(ip net.IP, msg *echonet_lite.ECHONETLiteMessage) (CallbackCompleteStatus, error) {
		device := echonet_lite.IPAndEOJ{IP: ip, EOJ: msg.SEOJ}
		if msg.ESV == echonet_lite.ESVGet_Res {
			return callback(device, true, msg.Properties, nil)
		}
		// Getは EDT=nilが失敗
		successProperties := make(echonet_lite.Properties, 0, len(msg.Properties))
		failedEPCs := make([]echonet_lite.EPCType, 0, len(msg.Properties))
		for _, p := range msg.Properties {
			if p.EDT != nil {
				successProperties = append(successProperties, p)
			} else {
				failedEPCs = append(failedEPCs, p.EPC)
			}
		}
		return callback(device, false, successProperties, failedEPCs)
	})
	return msg, key
}

func (s *Session) StartGetProperties(device echonet_lite.IPAndEOJ, EPCs []echonet_lite.EPCType, callback GetPropertiesCallbackFunc) (Key, error) {
	msg, key := s.prepareStartGetProperties(device, EPCs, callback)
	if err := s.sendMessage(device.IP, msg); err != nil {
		return Key{}, err
	}
	return key, nil
}

// StartGetPropertiesWithRetry は、プロパティ取得を行い、タイムアウトした場合は go routineで再試行する
func (s *Session) StartGetPropertiesWithRetry(ctx1 context.Context, device echonet_lite.IPAndEOJ, EPCs []echonet_lite.EPCType, callback GetPropertiesCallbackFunc) error {
	desc := fmt.Sprintf("StartGetPropertiesWithRetry(%v, %v)", device, EPCs)

	ctx, cancel := context.WithCancel(ctx1)

	msg, key := s.prepareStartGetProperties(device, EPCs, func(device echonet_lite.IPAndEOJ, success bool, properties echonet_lite.Properties, FailedEPCs []echonet_lite.EPCType) (CallbackCompleteStatus, error) {
		cancel()
		_, err := callback(device, success, properties, FailedEPCs)
		return CallbackFinished, err
	})

	err := s.sendMessage(device.IP, msg)
	if err != nil {
		cancel()
		s.UnregisterCallback(key)
		return err
	}

	go func() {
		defer cancel() // ゴルーチン終了時にキャンセル

		// 再送カウンタ
		retryCount := 0

		// 初回のタイマーをジッタ付きで作成
		intervalWithJitter := s.calculateRetryIntervalWithJitter(0)
		timer := time.NewTimer(intervalWithJitter)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				s.UnregisterCallback(key)

				if retryCount > 0 {
					slog.Info("リトライ後に完了", "desc", desc, "retryCount", retryCount)
				}
				return

			case <-timer.C:
				// タイムアウトした場合
				retryCount++

				if retryCount >= s.MaxRetries {
					// 最大再送回数に達した場合
					_ = s.notifyDeviceTimeout(device, 0)
					return
				}

				// 次の再送間隔をジッタ付きで計算 (retryCountをパラメータとして渡す)
				nextInterval := s.calculateRetryIntervalWithJitter(retryCount)

				// ログ出力（ジッタ付き間隔も表示）
				slog.Info("リクエストを再送します", "desc", desc, "retry", retryCount, "maxRetries", s.MaxRetries, "nextInterval", nextInterval)
				// fmt.Printf("%v: リクエストを再送します (試行 %d/%d)\n", desc, retryCount, s.MaxRetries) // DEBUG

				// 再送
				if err := s.sendMessage(device.IP, msg); err != nil {
					return
				}

				// タイマーをジッタ付き間隔でリセット
				timer.Reset(nextInterval)
			}
		}
	}()
	return nil
}

// notifyDeviceTimeout - デバイスタイムアウトを詳細情報付きで通知
func (s *Session) notifyDeviceTimeout(device echonet_lite.IPAndEOJ, totalDuration time.Duration) error {
	maxRetriesErr := ErrMaxRetriesReached{
		MaxRetries:    s.MaxRetries,
		Device:        device,
		TotalDuration: totalDuration,
		RetryInterval: s.RetryInterval,
	}
	if s.TimeoutCh != nil {
		select {
		case s.TimeoutCh <- SessionTimeoutEvent{
			Device: device,
			Type:   SessionTimeoutMaxRetries,
			Error:  maxRetriesErr,
		}:
			// 送信成功
			slog.Info("デバイスタイムアウト通知を送信", "device", device.Specifier(), "totalDuration", totalDuration)
		default:
			// チャンネルがブロックされている場合は無視
			slog.Warn("タイムアウト通知チャンネルがブロックされています", "device", device.Specifier())
		}
	} else {
		slog.Warn("タイムアウト通知チャンネルが設定されていません", "device", device.Specifier())
	}
	return maxRetriesErr
}

func (s *Session) CreateSetPropertyMessage(device echonet_lite.IPAndEOJ, properties echonet_lite.Properties) *echonet_lite.ECHONETLiteMessage {
	return &echonet_lite.ECHONETLiteMessage{
		TID:        s.newTID(),
		SEOJ:       s.eoj,
		DEOJ:       device.EOJ,
		ESV:        echonet_lite.ESVSetC,
		Properties: properties,
	}
}

// コールバックを登録解除する関数
func (s *Session) UnregisterCallback(key Key) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.dispatchTable, key)
}

// registerCallbackForResponse は応答受信用のコールバックを登録する
func (s *Session) registerCallbackForResponse(
	ctx context.Context,
	key Key,
	responseESVs []echonet_lite.ESVType,
	responseCh chan<- *echonet_lite.ECHONETLiteMessage,
) {
	s.registerCallback(key, responseESVs, func(ip net.IP, respMsg *echonet_lite.ECHONETLiteMessage) (CallbackCompleteStatus, error) {
		// 応答メッセージをチャネルに送信
		select {
		case <-ctx.Done():
			// コンテキストがキャンセルされた場合は何もしない
		default:
			responseCh <- respMsg
		}

		// 必ず登録解除する（ブロードキャストを想定しない）
		s.UnregisterCallback(key)

		return CallbackFinished, nil
	})
}

// deviceResponse はブロードキャスト時の各デバイスの応答状態を管理する
type deviceResponse struct {
	device     echonet_lite.IPAndEOJ
	responseCh chan *echonet_lite.ECHONETLiteMessage
	responded  bool
	response   *echonet_lite.ECHONETLiteMessage
}

// BroadcastResult はブロードキャストの結果を表す
type BroadcastResult struct {
	Device   echonet_lite.IPAndEOJ
	Response *echonet_lite.ECHONETLiteMessage
	Error    error
}

// waitForResponseWithRetry は送信後の応答待ちと再送処理を行う
// 注意: 最初の送信は呼び出し側で行うこと
func (s *Session) waitForResponseWithRetry(
	ctx context.Context,
	device echonet_lite.IPAndEOJ,
	msg *echonet_lite.ECHONETLiteMessage,
	responseCh <-chan *echonet_lite.ECHONETLiteMessage,
) (*echonet_lite.ECHONETLiteMessage, error) {
	// 再送カウンタと時間追跡
	retryCount := 0
	startTime := time.Now()

	// 初回のタイマーをジッタ付きで作成
	intervalWithJitter := s.calculateRetryIntervalWithJitter(0)
	timer := time.NewTimer(intervalWithJitter)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			// 親コンテキストがキャンセルされた場合
			return nil, ctx.Err()

		case respMsg := <-responseCh:
			if retryCount > 0 {
				slog.Info("リトライ後に完了", "device", device, "retryCount", retryCount)
			}
			// 応答を受信した場合
			return respMsg, nil

		case <-timer.C:
			// タイムアウトした場合
			retryCount++

			if retryCount >= s.MaxRetries {
				// 最大再送回数に達した場合
				totalDuration := time.Since(startTime)
				return nil, s.notifyDeviceTimeout(device, totalDuration)
			}

			// 次の再送間隔をジッタ付きで計算 (retryCountをパラメータとして渡す)
			nextInterval := s.calculateRetryIntervalWithJitter(retryCount)

			// ログ出力（ジッタ付き間隔も表示）
			slog.Info("リクエストを再送します", "device", device, "retry", retryCount+1, "maxRetries", s.MaxRetries, "nextInterval", nextInterval)

			// 再送
			if err := s.sendMessage(device.IP, msg); err != nil {
				return nil, fmt.Errorf("failed to resend message to device %v (retry %d/%d): %w", device, retryCount+1, s.MaxRetries, err)
			}

			// タイマーをジッタ付き間隔でリセット
			timer.Reset(nextInterval)
		}
	}
}

// collectBroadcastResults は各デバイスの応答結果を収集する
func (s *Session) collectBroadcastResults(deviceResponses []*deviceResponse) []BroadcastResult {
	results := make([]BroadcastResult, len(deviceResponses))
	for i, dr := range deviceResponses {
		result := BroadcastResult{
			Device: dr.device,
		}
		if dr.responded {
			result.Response = dr.response
		} else {
			result.Error = fmt.Errorf("no response received")
		}
		results[i] = result
	}
	return results
}

// 共通処理を行う内部関数
func (s *Session) sendRequestWithContext(
	ctx context.Context,
	device echonet_lite.IPAndEOJ,
	msg *echonet_lite.ECHONETLiteMessage,
) (*echonet_lite.ECHONETLiteMessage, error) {
	// 結果を受け取るためのチャネル
	responseCh := make(chan *echonet_lite.ECHONETLiteMessage, 1)

	// キーを取得
	key := MakeKey(msg)

	// コールバックを登録
	s.registerCallbackForResponse(ctx, key, msg.ESV.ResponseESVs(), responseCh)

	// 関数終了時にコールバックを登録解除するための遅延処理
	defer func() {
		// レスポンスチャネルがコールバック内で既に登録解除される可能性があるが、
		// 念のため再度呼び出す（重複しても問題ない）
		s.UnregisterCallback(key)
	}()

	// 最初のリクエスト送信
	if err := s.sendMessage(device.IP, msg); err != nil {
		return nil, fmt.Errorf("failed to send initial message to device %v: %w", device, err)
	}

	// 応答待ちと再送処理
	return s.waitForResponseWithRetry(ctx, device, msg, responseCh)
}

// sendRequestWithContextBroadcast は同一クラスコードの複数デバイスへブロードキャストリクエストを送信する
func (s *Session) sendRequestWithContextBroadcast(
	ctx context.Context,
	devices []echonet_lite.IPAndEOJ,
	msg *echonet_lite.ECHONETLiteMessage,
) ([]BroadcastResult, error) {
	if len(devices) == 0 {
		return nil, fmt.Errorf("devices list is empty")
	}

	// 全デバイスが同一IPとclassCodeを持つことを確認
	firstDevice := devices[0]
	for _, d := range devices[1:] {
		if !d.IP.Equal(firstDevice.IP) || d.EOJ.ClassCode() != firstDevice.EOJ.ClassCode() {
			return nil, fmt.Errorf("all devices must have same IP and classCode")
		}
	}

	// ブロードキャスト用メッセージを作成（instanceCode = 0）
	broadcastMsg := &echonet_lite.ECHONETLiteMessage{
		TID:              msg.TID,
		SEOJ:             msg.SEOJ,
		DEOJ:             echonet_lite.MakeEOJ(msg.DEOJ.ClassCode(), 0),
		ESV:              msg.ESV,
		Properties:       msg.Properties,
		SetGetProperties: msg.SetGetProperties,
	}

	// 各デバイス用の応答管理構造体を準備
	deviceResponses := make([]*deviceResponse, len(devices))
	deviceMap := make(map[echonet_lite.EOJ]*deviceResponse)
	for i, device := range devices {
		dr := &deviceResponse{
			device:     device,
			responseCh: make(chan *echonet_lite.ECHONETLiteMessage, 1),
			responded:  false,
		}
		deviceResponses[i] = dr
		deviceMap[device.EOJ] = dr
	}

	// TIDに対して1つのコールバックを登録（複数デバイスからの応答を各chanに分配）
	key := MakeKey(broadcastMsg)
	s.registerCallback(key, msg.ESV.ResponseESVs(), func(ip net.IP, respMsg *echonet_lite.ECHONETLiteMessage) (CallbackCompleteStatus, error) {
		// 応答したデバイスを特定し、対応するchanに送信
		if dr, ok := deviceMap[respMsg.SEOJ]; ok {
			select {
			case dr.responseCh <- respMsg:
				// 応答をチャネルに送信
			default:
				// チャネルがフルまたは既に閉じられている場合は無視
			}
		}
		return CallbackContinue, nil
	})

	// ブロードキャストメッセージを送信
	if err := s.sendMessage(devices[0].IP, broadcastMsg); err != nil {
		// エラー時はコールバック登録解除とチャネルクリーンアップ
		s.UnregisterCallback(key)
		for _, dr := range deviceResponses {
			close(dr.responseCh)
		}
		return nil, fmt.Errorf("failed to send broadcast message to %d devices at %v (class: %04X): %w", len(devices), devices[0].IP, uint16(devices[0].EOJ.ClassCode()), err)
	}

	// 各インスタンス用にgoroutineでwaitForResponseWithRetryを実行
	var wg sync.WaitGroup
	for i, dr := range deviceResponses {
		wg.Add(1)
		go func(index int, deviceResp *deviceResponse) {
			defer wg.Done()

			// 各デバイス用のメッセージを作成
			deviceMsg := &echonet_lite.ECHONETLiteMessage{
				TID:              msg.TID,
				SEOJ:             msg.SEOJ,
				DEOJ:             deviceResp.device.EOJ,
				ESV:              msg.ESV,
				Properties:       msg.Properties,
				SetGetProperties: msg.SetGetProperties,
			}

			// 各デバイス用の応答待ち（最初の送信は既に完了）
			respMsg, err := s.waitForResponseWithRetry(ctx, deviceResp.device, deviceMsg, deviceResp.responseCh)
			if err == nil {
				deviceResp.responded = true
				deviceResp.response = respMsg
			}
		}(i, dr)
	}

	// 全goroutineの完了を待つ
	wg.Wait()

	// 全goroutineが完了した後にクリーンアップ
	s.UnregisterCallback(key)
	for _, dr := range deviceResponses {
		close(dr.responseCh)
	}

	// 結果を収集
	return s.collectBroadcastResults(deviceResponses), nil
}

// GetPropertiesBroadcast - 同一クラスコードの複数デバイスからブロードキャストでプロパティ取得
func (s *Session) GetPropertiesBroadcast(
	ctx context.Context,
	devices []echonet_lite.IPAndEOJ,
	EPCs []echonet_lite.EPCType,
) ([]BroadcastResult, error) {
	if len(devices) == 0 {
		return nil, fmt.Errorf("devices list is empty")
	}

	// メッセージを作成（最初のデバイスをベースにする）
	msg := s.CreateGetPropertyMessage(devices[0], EPCs)

	// ブロードキャスト送信
	results, err := s.sendRequestWithContextBroadcast(ctx, devices, msg)
	if err != nil {
		return nil, err
	}

	// 各デバイスの結果を処理
	for i := range results {
		result := &results[i]
		if result.Response != nil {
			// 共通処理を使用
			success, successProperties, _ := s.processGetPropertiesResponse(result.Device, result.Response)

			// 結果が空の場合はエラーとして扱う
			if !success && len(successProperties) == 0 {
				result.Error = fmt.Errorf("no properties retrieved successfully")
			}
		}
	}

	return results, nil
}

// processGetPropertiesResponse は Get プロパティの応答を共通処理する
func (s *Session) processGetPropertiesResponse(device echonet_lite.IPAndEOJ, respMsg *echonet_lite.ECHONETLiteMessage) (bool, echonet_lite.Properties, []echonet_lite.EPCType) {
	// 応答を処理
	success := respMsg.ESV == echonet_lite.ESVGet_Res

	// 成功/失敗のプロパティを分類
	successProperties := make(echonet_lite.Properties, 0, len(respMsg.Properties))
	failedEPCs := make([]echonet_lite.EPCType, 0, len(respMsg.Properties))

	for _, p := range respMsg.Properties {
		if p.EDT != nil {
			successProperties = append(successProperties, p)
		} else {
			failedEPCs = append(failedEPCs, p.EPC)
		}
	}

	failedEPCs = s.updateFailedEPCs(device, successProperties, failedEPCs)

	return success, successProperties, failedEPCs
}

func (s *Session) updateFailedEPCs(device echonet_lite.IPAndEOJ, success echonet_lite.Properties, failed []echonet_lite.EPCType) []echonet_lite.EPCType {
	key := device.Key()

	s.mu.Lock()
	defer s.mu.Unlock()

	// 既存の失敗リストを取得 (存在しない場合は空のスライス)
	existingFailedEPCs := make([]echonet_lite.EPCType, 0)
	if f, ok := s.failedEPCs[key]; ok {
		existingFailedEPCs = append(existingFailedEPCs, f...) // コピーを作成
	}

	// 1. 今回成功したEPCを既存の失敗リストから削除する
	if len(success) > 0 {
		remainingFailedEPCs := make([]echonet_lite.EPCType, 0, len(existingFailedEPCs))
		successEPCs := make(map[echonet_lite.EPCType]struct{}, len(success))
		for _, p := range success {
			successEPCs[p.EPC] = struct{}{}
		}
		for _, epc := range existingFailedEPCs {
			if _, found := successEPCs[epc]; !found {
				remainingFailedEPCs = append(remainingFailedEPCs, epc)
			}
		}
		existingFailedEPCs = remainingFailedEPCs // 更新
	}

	// 2. 今回失敗したEPCのうち、まだ記録されていないものを追加し、戻り値リストを作成する
	newlyFailedForReturn := make([]echonet_lite.EPCType, 0, len(failed))
	if len(failed) > 0 {
		currentFailedSet := make(map[echonet_lite.EPCType]struct{}, len(existingFailedEPCs))
		for _, epc := range existingFailedEPCs {
			currentFailedSet[epc] = struct{}{}
		}

		for _, epc := range failed {
			if _, found := currentFailedSet[epc]; !found {
				// まだ記録されていない失敗
				existingFailedEPCs = append(existingFailedEPCs, epc)     // 内部状態に追加
				newlyFailedForReturn = append(newlyFailedForReturn, epc) // 戻り値リストに追加
				currentFailedSet[epc] = struct{}{}                       // Setにも追加して重複を防ぐ
			}
		}
	}

	// 3. 更新された失敗リストをマップに保存（空なら削除）
	if len(existingFailedEPCs) == 0 {
		delete(s.failedEPCs, key)
	} else {
		s.failedEPCs[key] = existingFailedEPCs
	}

	return newlyFailedForReturn // 今回新たに失敗として記録されたEPCのみを返す
}

// GetProperties - プロパティ取得
func (s *Session) GetProperties(
	ctx context.Context,
	device echonet_lite.IPAndEOJ,
	EPCs []echonet_lite.EPCType,
) (bool, echonet_lite.Properties, []echonet_lite.EPCType, error) {
	// メッセージを作成
	msg := s.CreateGetPropertyMessage(device, EPCs)

	// 共通処理を呼び出し
	respMsg, err := s.sendRequestWithContext(ctx, device, msg)

	// エラーチェック
	if err != nil {
		// タイムアウトやコンテキストキャンセルの場合
		return false, nil, EPCs, err
	}

	// 共通処理を使用
	success, successProperties, failedEPCs := s.processGetPropertiesResponse(device, respMsg)

	return success, successProperties, failedEPCs, nil
}

// SetProperties - プロパティ設定
func (s *Session) SetProperties(
	ctx context.Context,
	device echonet_lite.IPAndEOJ,
	properties echonet_lite.Properties,
) (bool, echonet_lite.Properties, []echonet_lite.EPCType, error) {
	// メッセージを作成
	msg := s.CreateSetPropertyMessage(device, properties)

	// 共通処理を呼び出し
	respMsg, err := s.sendRequestWithContext(ctx, device, msg)

	// エラーチェック
	if err != nil {
		// タイムアウトやコンテキストキャンセルの場合
		failedEPCs := make([]echonet_lite.EPCType, 0, len(properties))
		for _, p := range properties {
			failedEPCs = append(failedEPCs, p.EPC)
		}
		return false, nil, failedEPCs, err
	}

	// 応答を処理
	success := respMsg.ESV == echonet_lite.ESVSet_Res

	// 成功/失敗のプロパティを分類
	successProperties := make(echonet_lite.Properties, 0, len(properties))
	failedEPCs := make([]echonet_lite.EPCType, 0, len(properties))

	// Setは EDT == nil が成功
	for i, p := range respMsg.Properties {
		if p.EDT == nil {
			successProperties = append(successProperties, properties[i])
		} else {
			failedEPCs = append(failedEPCs, p.EPC)
		}
	}

	return success, successProperties, failedEPCs, nil
}
