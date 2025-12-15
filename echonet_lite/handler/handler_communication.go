package handler

import (
	"context"
	"crypto/rand"
	"echonet-list/echonet_lite"
	"fmt"
	"log/slog"
	"math/big"
	mathrand "math/rand"
	"net"
	"sync"
	"time"
)

// activeUpdateEntry はアクティブな更新処理のエントリ
type activeUpdateEntry struct {
	startTime time.Time          // 更新開始時刻
	cancel    context.CancelFunc // キャンセル関数（nilの場合もある）
}

// CommunicationHandler は、ECHONET Lite 通信機能を担当する構造体
type CommunicationHandler struct {
	session         *Session                      // セッション
	localDevices    DeviceProperties              // 自ノードが所有するデバイスのプロパティ
	dataAccessor    DataAccessor                  // データアクセス機能
	notifier        NotificationRelay             // 通知中継
	ctx             context.Context               // コンテキスト
	Debug           bool                          // デバッグモード
	activeUpdatesMu sync.RWMutex                  // アクティブな更新処理の排他制御
	activeUpdates   map[string]*activeUpdateEntry // IP+EOJ別のアクティブな更新処理 (key: "IP:ClassCode:InstanceCode")
}

// NewCommunicationHandler は、CommunicationHandlerの新しいインスタンスを作成する
func NewCommunicationHandler(
	ctx context.Context,
	session *Session,
	localDevices DeviceProperties,
	dataAccessor DataAccessor,
	notifier NotificationRelay,
	debug bool,
) *CommunicationHandler {
	h := &CommunicationHandler{
		session:       session,
		localDevices:  localDevices,
		dataAccessor:  dataAccessor,
		notifier:      notifier,
		ctx:           ctx,
		Debug:         debug,
		activeUpdates: make(map[string]*activeUpdateEntry),
	}

	// バックグラウンドクリーンアップを開始
	go h.startActiveUpdatesCleanup()

	return h
}

// SetDebug は、デバッグモードを設定する
func (h *CommunicationHandler) SetDebug(debug bool) {
	h.Debug = debug
	h.session.Debug = debug
}

// makeDeviceKey はデバイスのキー文字列を生成する
// 形式: "IP:ClassCode:InstanceCode" (例: "192.168.1.100:0291:01")
func makeDeviceKey(device IPAndEOJ) string {
	return fmt.Sprintf("%s:%04X:%02X", device.IP.String(), uint16(device.EOJ.ClassCode()), device.EOJ.InstanceCode())
}

// isUpdateActive は指定されたデバイス(IP+EOJ)の更新処理がアクティブかどうかを確認する
// force=trueの場合、既存の更新処理をキャンセルしてfalseを返す
func (h *CommunicationHandler) isUpdateActive(device IPAndEOJ, force bool) bool {
	deviceKey := makeDeviceKey(device)

	if force {
		// force=trueの場合、既存の更新処理をキャンセルする
		h.cancelExistingUpdate(deviceKey)
		return false
	}

	h.activeUpdatesMu.RLock()
	entry, exists := h.activeUpdates[deviceKey]
	h.activeUpdatesMu.RUnlock()

	if !exists {
		return false
	}

	// 古いエントリは無効とする
	if time.Since(entry.startTime) > MaxUpdateAge {
		h.activeUpdatesMu.Lock()
		// 再度チェックして、まだ古い場合のみ削除（Double-checked locking pattern）
		if entry2, exists2 := h.activeUpdates[deviceKey]; exists2 && time.Since(entry2.startTime) > MaxUpdateAge {
			if entry2.cancel != nil {
				entry2.cancel()
			}
			delete(h.activeUpdates, deviceKey)
		}
		h.activeUpdatesMu.Unlock()
		return false
	}

	return true
}

// cancelExistingUpdate は既存の更新処理をキャンセルする
func (h *CommunicationHandler) cancelExistingUpdate(deviceKey string) {
	h.activeUpdatesMu.Lock()
	defer h.activeUpdatesMu.Unlock()

	if entry, exists := h.activeUpdates[deviceKey]; exists {
		if entry.cancel != nil {
			slog.Debug("既存の更新処理をキャンセル", "device", deviceKey)
			entry.cancel()
		}
		delete(h.activeUpdates, deviceKey)
	}
}

// markUpdateActive は指定されたデバイス(IP+EOJ)の更新処理をアクティブとしてマークする
func (h *CommunicationHandler) markUpdateActive(device IPAndEOJ, cancel context.CancelFunc) {
	deviceKey := makeDeviceKey(device)
	h.activeUpdatesMu.Lock()
	h.activeUpdates[deviceKey] = &activeUpdateEntry{
		startTime: time.Now(),
		cancel:    cancel,
	}
	h.activeUpdatesMu.Unlock()
}

// markUpdateInactive は指定されたデバイス(IP+EOJ)の更新処理を非アクティブとしてマークする
func (h *CommunicationHandler) markUpdateInactive(device IPAndEOJ) {
	deviceKey := makeDeviceKey(device)
	h.activeUpdatesMu.Lock()
	delete(h.activeUpdates, deviceKey)
	h.activeUpdatesMu.Unlock()
}

// startActiveUpdatesCleanup は、古いアクティブ更新エントリを定期的にクリーンアップする
func (h *CommunicationHandler) startActiveUpdatesCleanup() {
	ticker := time.NewTicker(MaxUpdateAge / 2) // 安全なクリーンアップ間隔（5分間隔）
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.cleanupStaleActiveUpdates()
		}
	}
}

// cleanupStaleActiveUpdates は、古いアクティブ更新エントリを削除する
func (h *CommunicationHandler) cleanupStaleActiveUpdates() {
	now := time.Now()
	h.activeUpdatesMu.Lock()
	defer h.activeUpdatesMu.Unlock()

	for deviceKey, entry := range h.activeUpdates {
		if now.Sub(entry.startTime) > MaxUpdateAge {
			if entry.cancel != nil {
				entry.cancel()
			}
			delete(h.activeUpdates, deviceKey)
			slog.Debug("Cleaned up stale active update entry", "device", deviceKey, "age", now.Sub(entry.startTime))
		}
	}
}

// isNodeProfileOnline は、指定されたIPアドレスのNodeProfileObjectがオンラインかどうかを確認する
func (h *CommunicationHandler) isNodeProfileOnline(ip net.IP) bool {
	nodeProfile := IPAndEOJ{
		IP:  ip,
		EOJ: echonet_lite.NodeProfileObject,
	}
	return !h.dataAccessor.IsOffline(nodeProfile)
}

// NotifyNodeList は、自ノードのインスタンスリストを通知する
func (h *CommunicationHandler) NotifyNodeList() error {
	list := echonet_lite.InstanceListNotification(h.localDevices.GetInstanceList())
	return h.session.Broadcast(echonet_lite.NodeProfileObject, echonet_lite.ESVINF, Properties{*list.Property()})
}

// onReceiveMessage は、メッセージを受信したときのコールバック
func (h *CommunicationHandler) onReceiveMessage(ip net.IP, msg *echonet_lite.ECHONETLiteMessage) error {
	if msg == nil {
		return nil
	}

	if h.Debug {
		fmt.Printf("%v: メッセージを受信: SEOJ:%v, DEOJ:%v, ESV:%v Property: %v\n",
			ip, msg.SEOJ, msg.DEOJ, msg.ESV,
			msg.Properties.String(msg.DEOJ.ClassCode()),
		)
	}

	found := h.localDevices.FindEOJ(msg.DEOJ)
	if len(found) == 0 {
		// 許容できないDEOJをもつmsgは破棄
		return fmt.Errorf("デバイス %v が見つかりません", msg.DEOJ)
	}
	eoj := found[0] // 全デバイス要求だが最初の1つで返信する
	msg.DEOJ = eoj

	switch msg.ESV {
	case echonet_lite.ESVGet:
		responses, ok := h.localDevices.GetProperties(eoj, msg.Properties)

		ESV := echonet_lite.ESVGet_Res
		if !ok {
			ESV = echonet_lite.ESVGet_SNA
		}
		if h.Debug {
			fmt.Printf("  Getメッセージに対する応答: %v\n", responses) // DEBUG
		}
		return h.session.SendResponse(ip, msg, ESV, responses, nil)

	case echonet_lite.ESVSetC, echonet_lite.ESVSetI:
		responses, success := h.localDevices.SetProperties(eoj, msg.Properties)

		// プロパティ設定が成功した場合、アナウンス対象のプロパティをチェックしてINF通知を送信
		if success {
			h.sendAnnouncementForChangedProperties(eoj, msg.Properties)
		}

		if msg.ESV != echonet_lite.ESVSetI || !success {
			ESV := echonet_lite.ESVSetI_SNA
			if msg.ESV == echonet_lite.ESVSetC {
				if success {
					ESV = echonet_lite.ESVSet_Res
				} else {
					ESV = echonet_lite.ESVSetC_SNA
				}
			}
			if h.Debug {
				fmt.Printf("  %vメッセージに対する応答: %v\n", msg.ESV, responses) // DEBUG
			}
			return h.session.SendResponse(ip, msg, ESV, responses, nil)
		}

	case echonet_lite.ESVSetGet:
		setResult, setSuccess := h.localDevices.SetProperties(eoj, msg.Properties)
		getResult, getSuccess := h.localDevices.GetProperties(eoj, msg.SetGetProperties)
		success := setSuccess && getSuccess

		// プロパティ設定が成功した場合、アナウンス対象のプロパティをチェックしてINF通知を送信
		if setSuccess {
			h.sendAnnouncementForChangedProperties(eoj, msg.Properties)
		}

		ESV := echonet_lite.ESVSetGet_Res
		if !success {
			ESV = echonet_lite.ESVSetGet_SNA
		}
		if h.Debug {
			fmt.Printf("  SetGetメッセージに対する応答: set:%v, get:%v\n", setResult, getResult) // DEBUG
		}
		return h.session.SendResponse(ip, msg, ESV, setResult, getResult)

	case echonet_lite.ESVINF_REQ:
		result, success := h.localDevices.GetProperties(eoj, msg.Properties)
		if !success {
			// 不可応答を個別に返す
			return h.session.SendResponse(ip, msg, echonet_lite.ESVINF_REQ_SNA, result, nil)
		}
		return h.session.Broadcast(msg.DEOJ, echonet_lite.ESVINF, result)

	default:
		fmt.Printf("  未対応のESV: %v\n", msg.ESV) // DEBUG
	}
	return nil
}

// onInfMessage は、INFメッセージを受信したときのコールバック
func (h *CommunicationHandler) onInfMessage(ip net.IP, msg *echonet_lite.ECHONETLiteMessage) error {
	if msg == nil {
		slog.Warn("無効なINFメッセージを受信しました: nil")
		return nil // 処理は継続
	}

	// 自分自身からのmulticastメッセージは無視する
	if h.session.IsLocalIP(ip) {
		if h.Debug {
			slog.Debug("自分自身からのINFメッセージを無視", "ip", ip, "SEOJ", msg.SEOJ)
		}
		return nil
	}

	slog.Info("INFメッセージを受信", "ip", ip, "SEOJ", msg.SEOJ, "DEOJ", msg.DEOJ, "ESV", msg.ESV, "Properties", msg.Properties.String(msg.SEOJ.ClassCode()))
	// fmt.Printf("INFメッセージを受信: %v %v, DEOJ:%v\n", ip, msg.SEOJ, msg.DEOJ) // DEBUG

	// デバイスの生存確認を記録（リトライ中のタイムアウト判定に使用）
	sourceDevice := echonet_lite.IPAndEOJ{IP: ip, EOJ: msg.SEOJ}
	h.session.SignalDeviceAlive(sourceDevice)

	// DEOJ は instanceCode = 0 (ワイルドカード) の場合がある
	if found := h.localDevices.FindEOJ(msg.DEOJ); len(found) == 0 {
		// 許容できないDEOJをもつmsgは破棄
		return nil
	} else {
		// 全デバイス要求だが最初の1つで返信する
		msg.DEOJ = found[0]
	}

	defer func() {
		if msg.ESV == echonet_lite.ESVINFC {
			replyProps := make([]Property, 0, len(msg.Properties))
			// EDTをnilにする
			for _, p := range msg.Properties {
				replyProps = append(replyProps, Property{
					EPC: p.EPC,
					EDT: nil,
				})
			}
			// 応答を返す
			err := h.session.SendResponse(ip, msg, echonet_lite.ESVINFC_Res, replyProps, nil)
			if err != nil {
				slog.Error("INFメッセージに対する応答の送信に失敗", "err", err)
			}
		}
	}()

	if msg.SEOJ.ClassCode() == echonet_lite.NodeProfile_ClassCode {
		// ノードプロファイルオブジェクトからのメッセージ
		for _, p := range msg.Properties {
			switch p.EPC {
			case echonet_lite.EPC_NPO_SelfNodeInstanceListS:
				err := h.onSelfNodeInstanceListS(IPAndEOJ{IP: ip, EOJ: msg.SEOJ}, true, p)
				if err != nil {
					slog.Error("SelfNodeInstanceListSの処理中エラー", "err", err)
					return err
				}
			case echonet_lite.EPC_NPO_InstanceListNotification:
				iln := echonet_lite.DecodeInstanceListNotification(p.EDT)
				if iln == nil {
					slog.Warn("InstanceListNotificationのデコードに失敗", "EDT", p.EDT)
					return nil // 処理は継続
				}
				return h.onInstanceList(ip, echonet_lite.InstanceList(*iln))
			default:
				slog.Info("未処理のEPC", "EPC", p.EPC)
			}
		}
	} else {
		// その他のオブジェクトからのメッセージ

		// IPアドレスが未登録の場合、デバイス情報を取得
		if !h.dataAccessor.HasIP(ip) {
			slog.Info("未登録のIPアドレスからのメッセージ", "ip", ip)
			err := h.GetSelfNodeInstanceListS(ip, false)
			if err != nil {
				slog.Error("SelfNodeInstanceListSの取得に失敗", "err", err)
				return err
			}
		}

		device := IPAndEOJ{IP: ip, EOJ: msg.SEOJ}

		// 未知のデバイスの場合、プロパティマップを取得
		if !h.dataAccessor.IsKnownDevice(device) {
			err := h.GetGetPropertyMap(device)
			if err != nil {
				slog.Error("プロパティマップの取得に失敗", "err", err)
				return err
			}
		}

		// プロパティの通知を処理
		if len(msg.Properties) > 0 {
			// Propertyの通知 -> 値を更新する
			h.dataAccessor.RegisterProperties(device, msg.Properties)
			fmt.Printf("%s: Propertyの通知: %v %v\n",
				time.Now().Format(time.RFC3339),
				device,
				msg.Properties.String(device.EOJ.ClassCode()),
			)

			// デバイス情報を保存
			h.dataAccessor.SaveDeviceInfo()
			h.dataAccessor.SetOffline(device, false)
		}
	}
	return nil
}

// onSelfNodeInstanceListS は、SelfNodeInstanceListSプロパティを受信したときのコールバック
func (h *CommunicationHandler) onSelfNodeInstanceListS(device IPAndEOJ, success bool, p Property) error {
	if !success {
		return fmt.Errorf("SelfNodeInstanceListSプロパティの取得に失敗しました: %v", device)
	}

	if p.EPC != echonet_lite.EPC_NPO_SelfNodeInstanceListS {
		return fmt.Errorf("予期しないEPC: %v (期待値: %v)", p.EPC, echonet_lite.EPC_NPO_SelfNodeInstanceListS)
	}

	il := echonet_lite.DecodeSelfNodeInstanceListS(p.EDT)
	if il == nil {
		return fmt.Errorf("SelfNodeInstanceListSのデコードに失敗しました: %X", p.EDT)
	}
	return h.onInstanceList(device.IP, echonet_lite.InstanceList(*il))
}

// onInstanceList は、インスタンスリストを受信したときのコールバック
func (h *CommunicationHandler) onInstanceList(ip net.IP, il echonet_lite.InstanceList) error {
	// NodeProfileObjectも追加して取得する
	il = append(il, echonet_lite.NodeProfileObject)

	// 1. そのIPアドレスの既存デバイスを取得
	criteria := FilterCriteria{
		Device: DeviceSpecifier{IP: &ip},
	}
	existingDevices := h.dataAccessor.Filter(criteria).ListIPAndEOJ()

	// 2. 新しいインスタンスリストをセットに変換（高速な検索のため）
	newDeviceSet := make(map[string]struct{})
	for _, eoj := range il {
		device := IPAndEOJ{IP: ip, EOJ: eoj}
		newDeviceSet[device.Key()] = struct{}{}
	}

	// 3. 削除されたデバイスを検出
	var devicesToRemove []IPAndEOJ
	for _, existingDevice := range existingDevices {
		if existingDevice.IP.Equal(ip) {
			if _, exists := newDeviceSet[existingDevice.Key()]; !exists {
				devicesToRemove = append(devicesToRemove, existingDevice)
			}
		}
	}

	// 4. 削除されたデバイスを削除
	for _, device := range devicesToRemove {
		if err := h.dataAccessor.RemoveDevice(device); err != nil {
			slog.Warn("デバイスの削除に失敗", "device", device, "err", err)
		} else {
			slog.Info("デバイスを削除", "device", device)
		}
	}

	// 5. デバイスの登録（新規・既存両方）
	for _, eoj := range il {
		device := IPAndEOJ{IP: ip, EOJ: eoj}
		h.dataAccessor.RegisterDevice(device)

		// NodeProfileが有効なデバイスとして報告している場合、
		// オフライン状態をオンラインに復帰
		if h.dataAccessor.IsOffline(device) {
			slog.Info("NodeProfileからのインスタンスリストによりデバイスをオンラインに復帰",
				"device", device.Specifier())
			h.dataAccessor.SetOffline(device, false)
		}
	}

	// デバイス情報の保存
	h.dataAccessor.SaveDeviceInfo()

	// 各デバイスのプロパティマップを取得
	var e []error
	for _, eoj := range il {
		device := IPAndEOJ{IP: ip, EOJ: eoj}
		if err := h.GetGetPropertyMap(device); err != nil {
			e = append(e, fmt.Errorf("デバイス %v のプロパティ取得に失敗: %w", device, err))
		}
	}

	// エラーがあれば報告（ただし処理は継続）
	if len(e) > 0 {
		for _, err := range e {
			slog.Warn("警告", "err", err)
		}
	}

	return nil
}

// onGetPropertyMap は、GetPropertyMapプロパティを受信したときのコールバック
func (h *CommunicationHandler) onGetPropertyMap(device IPAndEOJ, success bool, properties Properties, _ []EPCType) (CallbackCompleteStatus, error) {
	if !success {
		slog.Warn("GetPropertyMapプロパティの取得に失敗しました", "device", device)
		return CallbackFinished, nil
	}

	p := properties[0]

	if p.EPC != echonet_lite.EPCGetPropertyMap {
		slog.Warn("予期しないEPC", "EPC", p.EPC, "expected", echonet_lite.EPCGetPropertyMap)
		return CallbackFinished, nil
	}

	props := echonet_lite.DecodePropertyMap(p.EDT)
	if props == nil {
		return CallbackFinished, echonet_lite.ErrInvalidPropertyMap{EDT: p.EDT}
	}

	// 取得するプロパティのリストを作成
	forGet := make([]EPCType, 0, len(props))
	for epc := range props {
		forGet = append(forGet, epc)
	}

	// プロパティが見つからない場合
	if len(forGet) == 0 {
		slog.Info("デバイスにプロパティが見つかりません", "EOJ", device.EOJ)
		return CallbackFinished, nil
	}

	// プロパティを取得
	err := h.session.StartGetPropertiesWithRetry(
		h.ctx,
		device,
		forGet,
		func(device IPAndEOJ, success bool, properties Properties, failedEPCs []EPCType) (CallbackCompleteStatus, error) {
			if !success {
				slog.Warn("プロパティ取得に失敗", "device", device, "failedEPCs", failedEPCs)
			}

			// プロパティを登録
			h.dataAccessor.RegisterProperties(device, properties)

			// デバイス情報を保存
			h.dataAccessor.SaveDeviceInfo()
			h.dataAccessor.SetOffline(device, false)

			return CallbackFinished, nil
		},
	)

	if err != nil {
		slog.Error("プロパティ取得リクエストの送信に失敗", "err", err)
	}

	return CallbackFinished, err
}

// GetSelfNodeInstanceListS は、SelfNodeInstanceListSプロパティを取得する
func (h *CommunicationHandler) GetSelfNodeInstanceListS(ip net.IP, isMulti bool) error {
	// broadcastの場合、1秒無通信で完了とする
	// タイマーを作る
	var timer *time.Timer
	idleTimeout := time.Duration(2 * time.Second)
	if isMulti {
		timer = time.NewTimer(idleTimeout)
		defer timer.Stop()
	}
	key, err := h.session.StartGetProperties(
		IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}, []EPCType{echonet_lite.EPC_NPO_SelfNodeInstanceListS},
		func(ie IPAndEOJ, b bool, p Properties, f []EPCType) (CallbackCompleteStatus, error) {
			var completeStatus CallbackCompleteStatus
			if isMulti {
				completeStatus = CallbackContinue
				timer.Reset(idleTimeout)
			} else {
				completeStatus = CallbackFinished
			}
			return completeStatus, h.onSelfNodeInstanceListS(ie, b, p[0])
		})
	if err != nil {
		return err
	}
	if isMulti {
		defer h.session.UnregisterCallback(key)

		select {
		case <-timer.C:
			return nil
		case <-h.ctx.Done():
			return h.ctx.Err()
		}
	}
	return err
}

// GetGetPropertyMap は、GetPropertyMapプロパティを取得する
func (h *CommunicationHandler) GetGetPropertyMap(device IPAndEOJ) error {
	return h.session.StartGetPropertiesWithRetry(h.ctx, device, []EPCType{echonet_lite.EPCGetPropertyMap}, h.onGetPropertyMap)
}

// Discover は、ECHONET Liteデバイスを検出する
func (h *CommunicationHandler) Discover() error {
	slog.Info("Starting device discovery")
	start := time.Now()

	err := h.GetSelfNodeInstanceListS(BroadcastIP, true)

	duration := time.Since(start)
	if err != nil {
		slog.Error("Device discovery failed", "duration", duration, "error", err)
		return err
	}

	slog.Info("Device discovery completed", "duration", duration)
	return nil
}

// GetProperties は、プロパティ値を取得する
// 成功時には ip, eoj と properties を返す
func (h *CommunicationHandler) GetProperties(device IPAndEOJ, EPCs []EPCType, skipValidation bool) (DeviceAndProperties, error) {
	// 結果を格納する変数
	var result DeviceAndProperties

	if !skipValidation {
		// 指定されたEPCがGetPropertyMapに含まれているか確認
		valid, invalidEPCs, err := h.validateEPCsInPropertyMap(device, EPCs, GetPropertyMap)
		if err != nil {
			return DeviceAndProperties{}, err
		}
		if !valid {
			return DeviceAndProperties{}, fmt.Errorf("%v: 以下のEPCはGetPropertyMapに含まれていません: %v", device, invalidEPCs)
		}
	}

	success, properties, failedEPCs, err := h.session.GetProperties(
		h.ctx,
		device,
		EPCs,
	)

	if err != nil {
		slog.Error("プロパティ取得に失敗", "device", device, "err", err)
		return DeviceAndProperties{}, fmt.Errorf("%v: プロパティ取得に失敗: %w", device, err)
	}

	// 成功したプロパティを登録（部分的な成功の場合も含む）
	if len(properties) > 0 {
		// プロパティの登録
		h.dataAccessor.RegisterProperties(device, properties)

		// デバイス情報を保存
		h.dataAccessor.SaveDeviceInfo()
		h.dataAccessor.SetOffline(device, false)
	}

	// 結果を設定
	result.Device = device
	result.Properties = properties

	// 全体の成功/失敗を判定
	if !success {
		slog.Warn("一部のプロパティ取得に失敗", "device", device, "failed_epcs", failedEPCs)
	}

	return result, nil
}

// SetProperties は、プロパティ値を設定する
func (h *CommunicationHandler) SetProperties(device IPAndEOJ, properties Properties) (DeviceAndProperties, error) {
	// 結果を格納する変数
	var result DeviceAndProperties

	// 指定されたEPCがSetPropertyMapに含まれているか確認
	// Propertiesから各EPCを抽出
	epcs := make([]EPCType, 0, len(properties))
	for _, prop := range properties {
		epcs = append(epcs, prop.EPC)
	}

	valid, invalidEPCs, err := h.validateEPCsInPropertyMap(device, epcs, SetPropertyMap)
	if err != nil {
		return DeviceAndProperties{}, err
	}
	if !valid {
		return DeviceAndProperties{}, fmt.Errorf("以下のEPCはSetPropertyMapに含まれていません: %v", invalidEPCs)
	}

	success, successProperties, failedEPCs, err := h.session.SetProperties(
		h.ctx,
		device,
		properties,
	)

	if err != nil {
		slog.Error("プロパティ設定に失敗", "device", device, "err", err)
		return DeviceAndProperties{}, fmt.Errorf("%v: プロパティ設定に失敗: %w", device, err)
	}

	// 成功したプロパティを登録（部分的な成功の場合も含む）
	if len(successProperties) > 0 {
		// プロパティの登録
		h.dataAccessor.RegisterProperties(device, successProperties)

		// デバイス情報を保存
		h.dataAccessor.SaveDeviceInfo()
		h.dataAccessor.SetOffline(device, false)
	}

	// 結果を設定
	result.Device = device
	result.Properties = successProperties

	// 全体の成功/失敗を判定
	if !success {
		slog.Warn("一部のプロパティ設定に失敗", "device", device, "failed_epcs", failedEPCs)
	}

	return result, nil
}

// UpdateProperties は、フィルタリングされたデバイスのプロパティキャッシュを更新する
// force が true の場合、最終更新時刻に関わらず強制的に更新する
func (h *CommunicationHandler) UpdateProperties(criteria FilterCriteria, force bool) error {
	start := time.Now()

	// フィルタリングを実行
	filtered := h.dataAccessor.Filter(criteria)

	// フィルタリング結果が空の場合
	if filtered.Len() == 0 {
		slog.Warn("No devices matched criteria", "criteria", criteria)
		return fmt.Errorf("条件に一致するデバイスが見つかりません")
	}

	// 全てのデバイスの更新完了を待つためのWaitGroup
	var wg sync.WaitGroup
	var errMutex sync.Mutex
	var firstErr error
	storeError := func(err error) {
		errMutex.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMutex.Unlock()
	}

	// 同一IPアドレスへのリクエストの管理
	ipRequestCounts := make(map[string]int)
	baseDelay := 50 * time.Millisecond // 基準遅延時間を短縮

	// デバイスをIP+classCodeでグループ化し、ブロードキャスト可能なものと個別処理が必要なものに分類
	deviceGroups := make(map[string][]IPAndEOJ) // key: "IP:classCode"
	for _, device := range filtered.ListIPAndEOJ() {
		key := fmt.Sprintf("%s:%04X", device.IP.String(), uint16(device.EOJ.ClassCode()))
		deviceGroups[key] = append(deviceGroups[key], device)
	}

	// ブロードキャスト対象グループと個別処理グループに分類
	var broadcastGroups [][]IPAndEOJ
	var individualDevices []IPAndEOJ

	for _, group := range deviceGroups {
		if len(group) > 1 {
			broadcastGroups = append(broadcastGroups, group)
		} else {
			individualDevices = append(individualDevices, group[0])
		}
	}

	if h.Debug {
		slog.Info("Device processing strategy",
			"broadcast_groups", len(broadcastGroups),
			"individual_devices", len(individualDevices))
	}

	// ブロードキャストグループの処理
	for _, group := range broadcastGroups {
		// ブロードキャストグループ内の任意のデバイスでアクティブチェック
		groupActive := false
		for _, device := range group {
			if h.isUpdateActive(device, force) {
				groupActive = true
				break
			}
		}
		if groupActive {
			slog.Info("ブロードキャストグループ内のデバイスが既にアクティブのためスキップ", "group_size", len(group))
			continue
		}

		wg.Add(1)
		go func(devices []IPAndEOJ, force bool) {
			defer wg.Done()

			// グループ固有のcontextを作成
			ctx, cancel := context.WithCancel(h.ctx)
			defer cancel()

			// グループ内の全デバイスをアクティブとしてマーク（処理開始前に実行）
			// 全デバイスが同じcontextを共有するため、いずれかのデバイスへの強制更新で
			// グループ全体がキャンセルされる（ブロードキャストは単一リクエストなので適切な動作）
			for _, device := range devices {
				h.markUpdateActive(device, cancel)
			}
			defer func() {
				for _, device := range devices {
					h.markUpdateInactive(device)
				}
			}()

			h.processBroadcastGroup(ctx, devices, force, &errMutex, &firstErr)
		}(group, force)
	}

	// 個別デバイスの処理
	for _, device := range individualDevices {
		// forceがfalseの場合、最終更新時刻をチェック
		if !force {
			lastUpdateTime := h.dataAccessor.GetLastUpdateTime(device)
			if !lastUpdateTime.IsZero() && time.Since(lastUpdateTime) < UpdateIntervalThreshold {
				// fmt.Printf("デバイス %v は最近更新されたためスキップします (最終更新: %v)\n", device, lastUpdateTime.Format(time.RFC3339))
				continue // 更新をスキップ
			}
			if h.dataAccessor.IsOffline(device) && !h.isNodeProfileOnline(device.IP) {
				continue // オフラインのデバイスはスキップ（ただし、NodeProfileがオンラインの場合は更新を試行）
			}
		}

		propMap, ok := h.tryGetPropertyMap(device)
		if !ok {
			continue
		}

		// 同じIPアドレスのデバイスに対して、ジッタ付き遅延を計算
		ipStr := device.IP.String()
		ipRequestCounts[ipStr]++
		requestIndex := ipRequestCounts[ipStr]

		// 遅延の計算
		delay := calculateRequestDelay(requestIndex, baseDelay)

		// デバイスの更新処理がアクティブかチェック
		if h.isUpdateActive(device, force) {
			slog.Info("デバイスの更新処理が既にアクティブのためスキップ", "device", device.Specifier())
			continue
		}

		// goroutineを起動する直前にカウンターを増やす
		wg.Add(1)

		// 各デバイスに対して並列処理を実行
		go func(device IPAndEOJ, propMap PropertyMap, delay time.Duration) {
			defer wg.Done()

			// デバイス固有のcontextを作成
			ctx, cancel := context.WithCancel(h.ctx)
			defer cancel()

			// デバイスをアクティブとしてマーク（遅延前に実行）
			h.markUpdateActive(device, cancel)
			defer h.markUpdateInactive(device)

			h.processIndividualDevice(ctx, device, propMap, delay, storeError)
		}(device, propMap, delay)
	}

	// 全てのデバイスの更新が完了するまで待つ
	slog.Debug("Waiting for all device updates to complete")
	wg.Wait()

	duration := time.Since(start)
	// エラーがあれば返す
	if firstErr != nil {
		slog.Error("UpdateProperties completed with errors", "deviceCount", filtered.Len(), "duration", duration, "first_error", firstErr)
		return firstErr
	}

	slog.Debug("UpdateProperties completed successfully", "deviceCount", filtered.Len(), "duration", duration)
	return nil
}

// calculateRequestDelay は同一IPアドレスへの連続リクエストに対する遅延を計算する
// requestIndex: リクエストの順序（1から始まる）
// baseDelay: 基準となる遅延時間
func calculateRequestDelay(requestIndex int, baseDelay time.Duration) time.Duration {
	// 1番目のリクエストには遅延なし
	if requestIndex <= 1 {
		return 0
	}

	// 2番目以降のリクエストには遅延を追加
	// 指数バックオフの要素を加える（ただし上限を設定）
	multiplier := min(requestIndex-1, MaxDelayMultiplier)
	baseDelayForDevice := baseDelay * time.Duration(multiplier)

	// ±30%のジッタを追加（crypto/randを使用）
	jitterRange := float64(baseDelayForDevice) * JitterPercentage

	// 安全な乱数生成
	randomBig, err := rand.Int(rand.Reader, big.NewInt(1<<32))
	var jitterFactor float64
	if err != nil {
		// フォールバック: 時刻ベースの乱数
		jitterFactor = mathrand.Float64()
	} else {
		jitterFactor = float64(randomBig.Int64()) / float64(1<<32)
	}

	jitter := (jitterFactor - 0.5) * 2 * jitterRange
	delay := time.Duration(float64(baseDelayForDevice) + jitter)

	// 最小遅延を保証
	minDelay := time.Duration(float64(baseDelay) * MinIntervalRatio)
	return max(delay, minDelay)
}

// processBroadcastGroup はブロードキャスト対象のデバイスグループを処理する
func (h *CommunicationHandler) processBroadcastGroup(ctx context.Context, devices []IPAndEOJ, force bool, errMutex *sync.Mutex, errPtr *error) {
	if len(devices) == 0 {
		return
	}

	// グループ内デバイスのアクティブマークはgoroutine開始時に既に実行済み
	firstDevice := devices[0]

	storeError := func(err error) {
		errMutex.Lock()
		if *errPtr == nil {
			*errPtr = err
		}
		errMutex.Unlock()
	}

	// キャンセルチェック
	select {
	case <-ctx.Done():
		slog.Debug("ブロードキャストグループの更新処理がキャンセルされました", "device", firstDevice.Specifier(), "reason", ctx.Err())
		return
	default:
	}

	// グループ内の各デバイスのプロパティマップを確認し、有効なデバイスのみでブロードキャスト
	validDevicesWithMaps := make([]IPAndEOJ, 0, len(devices))
	var sharedPropMap PropertyMap

	for _, device := range devices {
		propMap, ok := h.tryGetPropertyMap(device)
		if ok {
			validDevicesWithMaps = append(validDevicesWithMaps, device)
			if sharedPropMap == nil {
				sharedPropMap = propMap // 最初の有効なプロパティマップを使用
			}
		}
		// 失敗した場合は tryGetPropertyMap 内でオフライン設定済み
	}

	if len(validDevicesWithMaps) == 0 {
		return // 有効なデバイスが無い場合は処理終了
	}

	// 有効なデバイスのみでブロードキャスト処理を継続
	devices = validDevicesWithMaps
	propMap := sharedPropMap

	// forceがfalseの場合の事前チェック
	if !force {
		validDevices := make([]IPAndEOJ, 0, len(devices))
		for _, device := range devices {
			lastUpdateTime := h.dataAccessor.GetLastUpdateTime(device)
			if !lastUpdateTime.IsZero() && time.Since(lastUpdateTime) < UpdateIntervalThreshold {
				continue // 更新をスキップ
			}
			if h.dataAccessor.IsOffline(device) && !h.isNodeProfileOnline(device.IP) {
				continue // オフラインのデバイスはスキップ（ただし、NodeProfileがオンラインの場合は更新を試行）
			}
			validDevices = append(validDevices, device)
		}
		devices = validDevices

		if len(devices) == 0 {
			return // 処理対象デバイス無し
		}
	}

	// ブロードキャストでプロパティ取得
	results, err := h.session.GetPropertiesBroadcast(
		ctx,
		devices,
		propMap.EPCs(),
	)

	if err != nil {
		storeError(fmt.Errorf("ブロードキャスト取得に失敗: %w", err))
		return
	}

	// 各デバイスの結果を処理
	for _, result := range results {
		deviceName := h.dataAccessor.DeviceStringWithAlias(result.Device)

		if result.Error != nil {
			storeError(fmt.Errorf("%v のプロパティ取得に失敗: %w", deviceName, result.Error))
			continue
		}

		if result.Response != nil {
			// 成功したプロパティを分類
			var successProperties Properties
			for _, p := range result.Response.Properties {
				if p.EDT != nil {
					successProperties = append(successProperties, p)
				}
			}

			if len(successProperties) > 0 {
				h.dataAccessor.RegisterProperties(result.Device, successProperties)
				h.dataAccessor.SaveDeviceInfo()
				h.dataAccessor.SetOffline(result.Device, false)
			}
		}
	}

	if h.Debug {
		slog.Info("Broadcast group processed", "device_count", len(devices), "first_device", firstDevice.Specifier())
	}
}

// tryGetPropertyMap は指定されたデバイスのプロパティマップを取得し、見つからない場合はGetPropertyMapリクエストを送信します
func (h *CommunicationHandler) tryGetPropertyMap(device IPAndEOJ) (PropertyMap, bool) {
	propMap := h.dataAccessor.GetPropertyMap(device, GetPropertyMap)
	if propMap != nil {
		return propMap, true
	}

	// プロパティマップが見つからない場合は、まずGetPropertyMapリクエストを送信
	slog.Debug("プロパティマップが見つからないため、GetPropertyMapを取得", "device", device.Specifier())

	success, properties, _, err := h.session.GetProperties(
		h.ctx,
		device,
		[]echonet_lite.EPCType{echonet_lite.EPCGetPropertyMap},
	)

	if err != nil || !success || len(properties) == 0 {
		// GetPropertyMapの取得に失敗した場合は、デバイスをオフライン状態に設定
		if !h.dataAccessor.IsOffline(device) {
			slog.Info("GetPropertyMap取得に失敗したため、デバイスをオフライン状態に設定", "device", device.Specifier())
			h.dataAccessor.SetOffline(device, true)
		}
		return nil, false
	}

	// 取得したプロパティを登録
	h.dataAccessor.RegisterProperties(device, properties)
	h.dataAccessor.SaveDeviceInfo()

	// 再度プロパティマップを取得
	propMap = h.dataAccessor.GetPropertyMap(device, GetPropertyMap)
	if propMap == nil {
		slog.Warn("GetPropertyMapを取得したがプロパティマップの生成に失敗", "device", device.Specifier())
		return nil, false
	}

	return propMap, true
}

// processIndividualDevice は個別デバイスを処理する
func (h *CommunicationHandler) processIndividualDevice(ctx context.Context, device IPAndEOJ, propMap PropertyMap, delay time.Duration, storeError func(error)) {
	deviceName := h.dataAccessor.DeviceStringWithAlias(device)

	// 同じIPアドレスのデバイスに対して遅延を追加
	if delay > 0 {
		select {
		case <-ctx.Done():
			slog.Debug("個別デバイスの更新処理がキャンセルされました", "device", deviceName, "reason", ctx.Err())
			return
		case <-time.After(delay):
			// 遅延完了
		}
	}

	// デバイスのアクティブマークはgoroutine開始時に既に実行済み

	success, properties, failedEPCs, err := h.session.GetProperties(
		ctx,
		device,
		propMap.EPCs(),
	)

	if err != nil {
		storeError(fmt.Errorf("%v のプロパティ取得に失敗: %w", deviceName, err))
		return
	}

	var changed []ChangedProperty

	// 成功したプロパティを登録（部分的な成功の場合も含む）
	if len(properties) > 0 {
		changed = h.dataAccessor.RegisterProperties(device, properties)
		// デバイス情報を保存
		h.dataAccessor.SaveDeviceInfo()
		h.dataAccessor.SetOffline(device, false)
	}

	// 結果を記録
	if len(changed) > 0 {
		classCode := device.EOJ.ClassCode()
		changes := make([]string, len(changed))
		for i, p := range changed {
			changes[i] = p.StringForClass(classCode)
		}
	}

	// 全体の成功/失敗を判定
	if !success && len(failedEPCs) > 0 {
		epcNames := make([]string, len(failedEPCs))
		for i, epc := range failedEPCs {
			epcNames[i] = epc.StringForClass(device.EOJ.ClassCode())
		}
		slog.Warn("プロパティ取得に失敗", "device", deviceName, "failed_epcs", epcNames)
	}
}

// validateEPCsInPropertyMap は、指定されたEPCがプロパティマップに含まれているかを確認する
func (h *CommunicationHandler) validateEPCsInPropertyMap(device IPAndEOJ, epcs []EPCType, mapType PropertyMapType) (bool, []EPCType, error) {
	invalidEPCs := []EPCType{}

	// デバイスが存在するか確認
	if !h.dataAccessor.IsKnownDevice(device) {
		return false, invalidEPCs, fmt.Errorf("デバイスが見つかりません: %v", device)
	}

	// 各EPCがプロパティマップに含まれているか確認
	for _, epc := range epcs {
		if !h.dataAccessor.HasEPCInPropertyMap(device, mapType, epc) {
			invalidEPCs = append(invalidEPCs, epc)
		}
	}

	return len(invalidEPCs) == 0, invalidEPCs, nil
}

// sendAnnouncementForChangedProperties は、変更されたプロパティがアナウンス対象の場合にINF通知を送信する
func (h *CommunicationHandler) sendAnnouncementForChangedProperties(eoj EOJ, properties Properties) {
	var announcementProps Properties

	// 各プロパティがアナウンス対象かどうかチェック
	for _, prop := range properties {
		if h.localDevices.IsAnnouncementTarget(eoj, prop.EPC) {
			// アナウンス対象の場合、現在の値を取得してリストに追加
			currentProp, ok := h.localDevices.Get(eoj, prop.EPC)
			if ok {
				announcementProps = append(announcementProps, currentProp)
			}
		}
	}

	// アナウンス対象のプロパティがある場合、INF通知を送信
	if len(announcementProps) > 0 {
		if h.Debug {
			slog.Debug("アナウンス対象プロパティの変更を通知", "SEOJ", eoj, "Properties", announcementProps)
		}
		err := h.session.Broadcast(eoj, echonet_lite.ESVINF, announcementProps)
		if err != nil {
			slog.Error("INF通知の送信に失敗", "err", err)
		}
	}
}

// ProcessPropertyUpdateHooks は、プロパティ更新後の追加処理を実行する
func (h *CommunicationHandler) ProcessPropertyUpdateHooks(device IPAndEOJ, properties Properties) error {
	// NodeProfile の EPC_NPO_SelfNodeInstanceListS の場合、オフライン復帰処理を実行
	if device.EOJ == echonet_lite.NodeProfileObject {
		for _, prop := range properties {
			if prop.EPC == echonet_lite.EPC_NPO_SelfNodeInstanceListS {
				return h.processInstanceListFromProperty(device, prop)
			}
		}
	}

	// 将来の追加処理もここに追加可能
	// if device.EOJ.ClassCode() == 0x0130 && prop.EPC == 0x80 { ... }

	return nil
}

// processInstanceListFromProperty は、プロパティからインスタンスリストを抽出してオフライン復帰処理を実行する
func (h *CommunicationHandler) processInstanceListFromProperty(device IPAndEOJ, property Property) error {
	// プロパティからInstanceListを抽出
	il := echonet_lite.DecodeSelfNodeInstanceListS(property.EDT)
	if il == nil {
		return fmt.Errorf("SelfNodeInstanceListSのデコードに失敗しました: %X", property.EDT)
	}

	// オフライン復帰処理のみを実行（無限ループを避けるため、プロパティ取得は行わない）
	return h.processInstanceListForOfflineRecovery(device.IP, echonet_lite.InstanceList(*il))
}

// processInstanceListForOfflineRecovery は、インスタンスリストによるオフライン復帰処理のみを行う
// onInstanceListと違い、プロパティマップ取得は行わない（無限ループ防止）
// テスト用にパッケージ内から呼び出し可能
func (h *CommunicationHandler) processInstanceListForOfflineRecovery(ip net.IP, il echonet_lite.InstanceList) error {
	// NodeProfileObjectも追加して取得する
	il = append(il, echonet_lite.NodeProfileObject)

	// 5. デバイスの登録（新規・既存両方）とオフライン復帰処理
	for _, eoj := range il {
		device := IPAndEOJ{IP: ip, EOJ: eoj}
		h.dataAccessor.RegisterDevice(device)

		// NodeProfileが有効なデバイスとして報告している場合、
		// オフライン状態をオンラインに復帰
		if h.dataAccessor.IsOffline(device) {
			slog.Info("NodeProfileからのインスタンスリストによりデバイスをオンラインに復帰",
				"device", device.Specifier())
			h.dataAccessor.SetOffline(device, false)
		}
	}

	// デバイス情報の保存
	h.dataAccessor.SaveDeviceInfo()

	// プロパティマップ取得は行わない（無限ループ防止のため）
	return nil
}
