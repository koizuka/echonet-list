package echonet_lite

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

const (
	DeviceFileName        = "devices.json"
	DeviceAliasesFileName = "aliases.json"

	CommandTimeout = 3 * time.Second // コマンド実行のタイムアウト時間
)

// ECHONETLiteHandler は、ECHONET Lite の通信処理を担当する構造体
type ECHONETLiteHandler struct {
	session       *Session
	devices       Devices
	DeviceAliases *DeviceAliases
	localDevices  DeviceProperties
	Debug         bool
	ctx           context.Context    // コンテキスト
	cancel        context.CancelFunc // コンテキストのキャンセル関数
}

// saveDeviceInfo は、デバイス情報をファイルに保存する共通処理
func (h *ECHONETLiteHandler) saveDeviceInfo() {
	if err := h.devices.SaveToFile(DeviceFileName); err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Log("警告: デバイス情報の保存に失敗しました: %v", err)
		}
		// 保存に失敗しても処理は継続
	}
}

// SetDebug は、デバッグモードを設定する
func (h *ECHONETLiteHandler) SetDebug(debug bool) {
	h.Debug = debug
	h.session.Debug = debug
}

// IsDebug は、現在のデバッグモードを返す
func (h *ECHONETLiteHandler) IsDebug() bool {
	return h.Debug
}

// NewECHONETLiteHandler は、ECHONETLiteHandler の新しいインスタンスを作成する
func NewECHONETLiteHandler(ctx context.Context, ip net.IP, seoj EOJ, debug bool) (*ECHONETLiteHandler, error) {
	// タイムアウト付きのコンテキストを作成
	handlerCtx, cancel := context.WithCancel(ctx)

	// 自ノードのセッションを作成
	session, err := CreateSession(handlerCtx, ip, seoj, debug)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, fmt.Errorf("接続に失敗: %w", err)
	}

	// デバイス情報を管理するオブジェクトを作成
	devices := NewDevices()

	// DeviceFileName のファイルが存在するなら読み込む
	if _, err := os.Stat(DeviceFileName); err == nil {
		err = devices.LoadFromFile(DeviceFileName)
		if err != nil {
			cancel() // エラーの場合はコンテキストをキャンセル
			return nil, fmt.Errorf("デバイス情報の読み込みに失敗: %w", err)
		}
	}

	aliases := NewDeviceAliases()

	// DeviceAliasesFileName のファイルが存在するなら読み込む
	if _, err := os.Stat(DeviceAliasesFileName); err == nil {
		err = aliases.LoadFromFile(DeviceAliasesFileName)
		if err != nil {
			cancel() // エラーの場合はコンテキストをキャンセル
			return nil, fmt.Errorf("エイリアス情報の読み込みに失敗: %w", err)
		}
	}

	localDevices := make(DeviceProperties)
	operationStatusOn := OperationStatus(true)
	manufacturerCode := ManufacturerCodeExperimental
	identificationNumber := IdentificationNumber{
		ManufacturerCode: manufacturerCode,
		UniqueIdentifier: make([]byte, 13), // 識別番号未設定は13バイトの0
	}

	err = localDevices.Set(NodeProfileObject1,
		&operationStatusOn,
		&identificationNumber,
		&manufacturerCode,
		&ECHONETLite_Version,
	)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, err
	}

	err = localDevices.Set(seoj,
		&operationStatusOn,
		&identificationNumber,
		&manufacturerCode,
	)
	if err != nil {
		cancel()
		return nil, err
	}

	err = localDevices.UpdateProfileObjectProperties()
	if err != nil {
		cancel()
		return nil, err
	}

	// 最後にやること
	localDevices.UpdateProfileObjectProperties()

	handler := &ECHONETLiteHandler{
		session:       session,
		devices:       devices,
		DeviceAliases: aliases,
		localDevices:  localDevices,
		Debug:         debug,
		ctx:           handlerCtx,
		cancel:        cancel,
	}

	// INFメッセージのコールバックを設定
	session.OnInf(handler.onInfMessage)
	session.OnReceive(handler.onReceiveMessage)

	return handler, nil
}

// Close は、ECHONETLiteHandler のリソースを解放する
func (h *ECHONETLiteHandler) Close() error {
	// コンテキストをキャンセル
	if h.cancel != nil {
		h.cancel()
	}
	return h.session.Close()
}

// StartMainLoop は、メインループを開始する
func (h *ECHONETLiteHandler) StartMainLoop() {
	go h.session.MainLoop()
}

func (h *ECHONETLiteHandler) NotifyNodeList() error {
	list := InstanceListNotification(h.localDevices.GetInstanceList())
	return h.session.Broadcast(NodeProfileObject1, ESVINF, Properties{*list.Property()})
}

func (h *ECHONETLiteHandler) onReceiveMessage(ip net.IP, msg *ECHONETLiteMessage) error {
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
	case ESVGet:
		responses, ok := h.localDevices.GetProperties(eoj, msg.Properties)

		ESV := ESVGet_Res
		if !ok {
			ESV = ESVGet_SNA
		}
		if h.Debug {
			fmt.Printf("  Getメッセージに対する応答: %v\n", responses) // DEBUG
		}
		return h.session.SendResponse(ip, msg, ESV, responses, nil)

	case ESVSetC, ESVSetI:
		responses, success := h.localDevices.SetProperties(eoj, msg.Properties)

		if msg.ESV != ESVSetI || !success {
			ESV := ESVSetI_SNA
			if msg.ESV == ESVSetC {
				if success {
					ESV = ESVSet_Res
				} else {
					ESV = ESVSetC_SNA
				}
			}
			if h.Debug {
				fmt.Printf("  %vメッセージに対する応答: %v\n", msg.ESV, responses) // DEBUG
			}
			return h.session.SendResponse(ip, msg, ESV, responses, nil)
		}

	case ESVSetGet:
		setResult, setSuccess := h.localDevices.SetProperties(eoj, msg.Properties)
		getResult, getSuccess := h.localDevices.GetProperties(eoj, msg.SetGetProperties)
		success := setSuccess && getSuccess

		ESV := ESVSetGet_Res
		if !success {
			ESV = ESVSetGet_SNA
		}
		if h.Debug {
			fmt.Printf("  SetGetメッセージに対する応答: set:%v, get:%v\n", setResult, getResult) // DEBUG
		}
		return h.session.SendResponse(ip, msg, ESV, setResult, getResult)

	case ESVINF_REQ:
		result, success := h.localDevices.GetProperties(eoj, msg.Properties)
		if !success {
			// 不可応答を個別に返す
			return h.session.SendResponse(ip, msg, ESVINF_REQ_SNA, result, nil)
		}
		return h.session.Broadcast(msg.DEOJ, ESVINF, result)

	default:
		fmt.Printf("  未対応のESV: %v\n", msg.ESV) // DEBUG
	}
	return nil
}

func (h *ECHONETLiteHandler) registerProperties(device IPAndEOJ, properties Properties) {
	h.devices.RegisterProperties(device, properties)
	if property, ok := properties.FindEPC(EPCIdentificationNumber); ok {
		if id := DecodeIdentificationNumber(property.EDT); id != nil {
			err := h.DeviceAliases.RegisterDeviceIdentification(device, id)
			if err != nil {
				if logger := GetLogger(); logger != nil {
					logger.Log("警告: IdentificationNumberの登録に失敗: %v", err)
				}
			}
		}
	}
}

// onInfMessage は、INFメッセージを受信したときのコールバック
func (h *ECHONETLiteHandler) onInfMessage(ip net.IP, msg *ECHONETLiteMessage) error {
	logger := GetLogger()
	if msg == nil {
		if logger != nil {
			logger.Log("警告: 無効なINFメッセージを受信しました: nil")
		}
		return nil // 処理は継続
	}

	if logger != nil {
		logger.Log("INFメッセージを受信: %v, SEOJ:%v, DEOJ:%v", ip, msg.SEOJ, msg.DEOJ)
	}

	// DEOJ は instanceCode = 0 (ワイルドカード) の場合がある
	if found := h.localDevices.FindEOJ(msg.DEOJ); len(found) == 0 {
		// 許容できないDEOJをもつmsgは破棄
		return nil
	} else {
		// 全デバイス要求だが最初の1つで返信する
		msg.DEOJ = found[0]
	}

	defer func() {
		if msg.ESV == ESVINFC {
			replyProps := make([]Property, 0, len(msg.Properties))
			// EDTをnilにする
			for _, p := range msg.Properties {
				replyProps = append(replyProps, Property{
					EPC: p.EPC,
					EDT: nil,
				})
			}
			// 応答を返す
			err := h.session.SendResponse(ip, msg, ESVINFC_Res, replyProps, nil)
			if err != nil {
				if logger != nil {
					logger.Log("エラー: INFメッセージに対する応答の送信に失敗: %v", err)
				}
			}
		}
	}()

	if msg.SEOJ.ClassCode() == NodeProfile_ClassCode {
		// ノードプロファイルオブジェクトからのメッセージ
		for _, p := range msg.Properties {
			switch p.EPC {
			case EPC_NPO_SelfNodeInstanceListS:
				err := h.onSelfNodeInstanceListS(IPAndEOJ{ip, msg.SEOJ}, true, p)
				if err != nil {
					if logger != nil {
						logger.Log("エラー: SelfNodeInstanceListSの処理中: %v", err)
					}
					return err
				}
			case EPC_NPO_InstanceListNotification:
				iln := DecodeInstanceListNotification(p.EDT)
				if iln == nil {
					if logger != nil {
						logger.Log("警告: InstanceListNotificationのデコードに失敗: %X", p.EDT)
					}
					return nil // 処理は継続
				}
				return h.onInstanceList(ip, InstanceList(*iln))
			default:
				if logger != nil {
					logger.Log("情報: 未処理のEPC: %v", p.EPC)
				}
			}
		}
	} else {
		// その他のオブジェクトからのメッセージ

		// IPアドレスが未登録の場合、デバイス情報を取得
		if !h.devices.HasIP(ip) {
			if logger != nil {
				logger.Log("情報: 未登録のIPアドレスからのメッセージ: %v", ip)
			}
			err := h.GetSelfNodeInstanceListS(ip)
			if err != nil {
				if logger != nil {
					logger.Log("エラー: SelfNodeInstanceListSの取得に失敗: %v", err)
				}
				return err
			}
		}

		device := IPAndEOJ{ip, msg.SEOJ}

		// 未知のデバイスの場合、プロパティマップを取得
		if !h.devices.IsKnownDevice(device) {
			if logger != nil {
				logger.Log("情報: 新しいデバイスを検出: %v", device)
			}
			fmt.Printf("新しいデバイスが検出されました: %v\n", device)
			err := h.GetGetPropertyMap(device)
			if err != nil {
				if logger != nil {
					logger.Log("エラー: プロパティマップの取得に失敗: %v", err)
				}
				return err
			}
		}

		// プロパティの通知を処理
		if len(msg.Properties) > 0 {
			// Propertyの通知 -> 値を更新する
			h.registerProperties(device, msg.Properties)
			fmt.Printf("%v: Propertyの通知: %v\n", device, msg.Properties)

			// デバイス情報を保存
			h.saveDeviceInfo()
		}
	}
	return nil
}

// onSelfNodeInstanceListS は、SelfNodeInstanceListSプロパティを受信したときのコールバック
func (h *ECHONETLiteHandler) onSelfNodeInstanceListS(device IPAndEOJ, success bool, p Property) error {
	if !success {
		return fmt.Errorf("SelfNodeInstanceListSプロパティの取得に失敗しました: %v", device)
	}

	if p.EPC != EPC_NPO_SelfNodeInstanceListS {
		return fmt.Errorf("予期しないEPC: %v (期待値: %v)", p.EPC, EPC_NPO_SelfNodeInstanceListS)
	}

	il := DecodeSelfNodeInstanceListS(p.EDT)
	if il == nil {
		return fmt.Errorf("SelfNodeInstanceListSのデコードに失敗しました: %X", p.EDT)
	}
	return h.onInstanceList(device.IP, InstanceList(*il))
}

func (h *ECHONETLiteHandler) onInstanceList(ip net.IP, il InstanceList) error {
	// デバイスの登録
	for _, eoj := range il {
		h.devices.RegisterDevice(IPAndEOJ{ip, eoj})
	}

	// デバイス情報の保存
	h.saveDeviceInfo()

	// 各デバイスのプロパティマップを取得
	var errors []error
	for _, eoj := range il {
		device := IPAndEOJ{ip, eoj}
		if err := h.GetGetPropertyMap(device); err != nil {
			errors = append(errors, fmt.Errorf("デバイス %v のプロパティ取得に失敗: %w", device, err))
		}
	}

	// エラーがあれば報告（ただし処理は継続）
	if len(errors) > 0 {
		for _, err := range errors {
			if logger := GetLogger(); logger != nil {
				logger.Log("警告: %v", err)
			}
		}
	}

	return nil
}

// onGetPropertyMap は、GetPropertyMapプロパティを受信したときのコールバック
func (h *ECHONETLiteHandler) onGetPropertyMap(device IPAndEOJ, success bool, properties Properties, _ []EPCType) (CallbackCompleteStatus, error) {
	logger := GetLogger()
	if !success {
		if logger != nil {
			logger.Log("警告: GetPropertyMapプロパティの取得に失敗しました: %v", device)
		}
		return CallbackFinished, nil
	}

	p := properties[0]

	if p.EPC != EPCGetPropertyMap {
		if logger != nil {
			logger.Log("警告: 予期しないEPC: %v (期待値: %v)", p.EPC, EPCGetPropertyMap)
		}
		return CallbackFinished, nil
	}

	props := DecodePropertyMap(p.EDT)
	if props == nil {
		return CallbackFinished, ErrInvalidPropertyMap{EDT: p.EDT}
	}

	// 取得するプロパティのリストを作成
	forGet := make([]EPCType, 0, len(props))
	for epc := range props {
		forGet = append(forGet, epc)
	}

	// プロパティが見つからない場合
	if len(forGet) == 0 {
		if logger != nil {
			logger.Log("情報: デバイス %v にプロパティが見つかりませんでした", device.EOJ)
		}
		return CallbackFinished, nil
	}

	// プロパティを取得
	err := h.session.GetProperties(
		device,
		forGet,
		func(device IPAndEOJ, success bool, properties Properties, failedEPCs []EPCType) (CallbackCompleteStatus, error) {
			if !success {
				if logger != nil {
					logger.Log("警告: プロパティ取得に失敗しました: %v, Failed EPCs: %v", device, failedEPCs)
				}
			}

			// プロパティを登録
			h.registerProperties(device, properties)

			// デバイス情報を保存
			h.saveDeviceInfo()

			return CallbackFinished, nil
		},
	)

	if err != nil {
		if logger != nil {
			logger.Log("エラー: プロパティ取得リクエストの送信に失敗: %v", err)
		}
	}

	return CallbackFinished, err
}

// GetSelfNodeInstanceListS は、SelfNodeInstanceListSプロパティを取得する
func (h *ECHONETLiteHandler) GetSelfNodeInstanceListS(ip net.IP) error {
	return h.session.GetProperties(
		IPAndEOJ{ip, NodeProfileObject1}, []EPCType{EPC_NPO_SelfNodeInstanceListS},
		func(ie IPAndEOJ, b bool, p Properties, f []EPCType) (CallbackCompleteStatus, error) {
			return CallbackFinished, h.onSelfNodeInstanceListS(ie, b, p[0])
		})
}

// GetGetPropertyMap は、GetPropertyMapプロパティを取得する
func (h *ECHONETLiteHandler) GetGetPropertyMap(device IPAndEOJ) error {
	return h.session.GetProperties(device, []EPCType{EPCGetPropertyMap}, h.onGetPropertyMap)
}

// Discover は、ECHONET Liteデバイスを検出する
func (h *ECHONETLiteHandler) Discover() error {
	return h.GetSelfNodeInstanceListS(BroadcastIP)
}

// ListDevices は、検出されたデバイスの一覧を表示する
func (h *ECHONETLiteHandler) ListDevices(criteria FilterCriteria) []DevicePropertyData {
	// フィルタリングを実行
	filtered := h.devices.Filter(criteria)

	return filtered.ListDevicePropertyData()
}

// validateEPCsInPropertyMap は、指定されたEPCがプロパティマップに含まれているかを確認する
func (h *ECHONETLiteHandler) validateEPCsInPropertyMap(device IPAndEOJ, epcs []EPCType, mapType PropertyMapType) (bool, []EPCType, error) {
	invalidEPCs := []EPCType{}

	// デバイスが存在するか確認
	if !h.devices.IsKnownDevice(device) {
		return false, invalidEPCs, fmt.Errorf("デバイスが見つかりません: %v", device)
	}

	// 各EPCがプロパティマップに含まれているか確認
	for _, epc := range epcs {
		if !h.devices.HasEPCInPropertyMap(device, mapType, epc) {
			invalidEPCs = append(invalidEPCs, epc)
		}
	}

	return len(invalidEPCs) == 0, invalidEPCs, nil
}

type DeviceAndProperties struct {
	Device     IPAndEOJ
	Properties Properties
}

// GetProperties は、プロパティ値を取得する
// 成功時には ip, eoj と properties を返す
func (h *ECHONETLiteHandler) GetProperties(device IPAndEOJ, EPCs []EPCType) (DeviceAndProperties, error) {
	// 結果を格納する変数
	var result DeviceAndProperties

	logger := GetLogger()

	// 指定されたEPCがGetPropertyMapに含まれているか確認
	valid, invalidEPCs, err := h.validateEPCsInPropertyMap(device, EPCs, GetPropertyMap)
	if err != nil {
		return DeviceAndProperties{}, err
	}
	if !valid {
		return DeviceAndProperties{}, fmt.Errorf("以下のEPCはGetPropertyMapに含まれていません: %v", invalidEPCs)
	}

	// コマンドを実行
	err = waitCallbackDone(h.ctx, func(callbackDone chan struct{}) error {
		// コンテキストは親コンテキスト（h.ctx）を使用
		err := h.session.GetProperties(device, EPCs, func(device IPAndEOJ, success bool, properties Properties, failedEPCs []EPCType) (CallbackCompleteStatus, error) {
			defer close(callbackDone) // 必ず完了を通知

			if !success {
				errMsg := fmt.Sprintf("プロパティ取得失敗: %v: %v", device, failedEPCs)
				if logger != nil {
					logger.Log("エラー: %s", errMsg)
				}

				return CallbackFinished, errors.New(errMsg)
			}

			// 成功した場合の処理
			result.Device = device
			result.Properties = properties

			// プロパティの登録
			h.registerProperties(device, properties)

			// デバイス情報を保存
			h.saveDeviceInfo()

			return CallbackFinished, nil
		})

		if err != nil {
			if logger != nil {
				logger.Log("エラー: プロパティ取得リクエストの送信に失敗: %v", err)
			}
			err = fmt.Errorf("プロパティ取得リクエストの送信に失敗: %w", err)
			close(callbackDone) // エラー時も完了を通知
		}

		return err
	}, "プロパティ取得がタイムアウトしました")

	return result, err
}

// SetProperties は、プロパティ値を設定する
func (h *ECHONETLiteHandler) SetProperties(device IPAndEOJ, properties Properties) (DeviceAndProperties, error) {
	// 結果を格納する変数
	var result DeviceAndProperties

	logger := GetLogger()

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

	// コマンドを実行
	err = waitCallbackDone(h.ctx, func(callbackDone chan struct{}) error {
		// コンテキストは親コンテキスト（h.ctx）を使用
		err := h.session.SetProperties(device, properties, func(device IPAndEOJ, success bool, returnedProperties Properties, failedEPCs []EPCType) (CallbackCompleteStatus, error) {
			defer close(callbackDone) // 必ず完了を通知

			if !success {
				errMsg := fmt.Sprintf("プロパティ設定失敗: %v: %v", device, failedEPCs)
				if logger != nil {
					logger.Log("エラー: %s", errMsg)
				}
				return CallbackFinished, errors.New(errMsg)
			}

			// 成功した場合の処理
			result.Device = device
			result.Properties = properties

			h.registerProperties(device, properties)

			// デバイス情報を保存
			h.saveDeviceInfo()

			return CallbackFinished, nil
		})

		if err != nil {
			if logger != nil {
				logger.Log("エラー: プロパティ設定リクエストの送信に失敗: %v", err)
			}
			err = fmt.Errorf("プロパティ設定リクエストの送信に失敗: %w", err)
			close(callbackDone) // エラー時も完了を通知
		}

		return err
	}, "プロパティ設定がタイムアウトしました")

	return result, err
}

// UpdateProperties は、フィルタリングされたデバイスのプロパティキャッシュを更新する
func (h *ECHONETLiteHandler) UpdateProperties(criteria FilterCriteria) error {
	// フィルタリングを実行
	filtered := h.devices.Filter(criteria)

	// フィルタリング結果が空の場合
	if filtered.Len() == 0 {
		return fmt.Errorf("条件に一致するデバイスが見つかりません")
	}

	fmt.Println("更新対象のデバイス:")
	for _, d := range filtered.ListIPAndEOJ() {
		fmt.Println("  ", d)
	}
	fmt.Println("プロパティの更新を開始します...")

	// タイムアウト付きのコンテキストを作成（親コンテキストを使用）
	timeoutCtx, cancel := context.WithTimeout(h.ctx, CommandTimeout)
	defer cancel()

	// 全てのデバイスの更新完了を待つためのWaitGroup
	var wg sync.WaitGroup
	var errMutex sync.Mutex
	var firstErr error

	// 各デバイスに対して処理を実行
	for _, device := range filtered.ListIPAndEOJ() {
		wg.Add(1)

		propMap := h.devices.GetPropertyMap(device, GetPropertyMap)
		if propMap == nil {
			errMutex.Lock()
			if firstErr == nil {
				firstErr = fmt.Errorf("プロパティマップが見つかりません: %v", device)
			}
			errMutex.Unlock()
			wg.Done()
			continue
		}

		// コールバック完了を待つためのチャネル
		callbackDone := make(chan struct{})

		err := h.session.GetProperties(device, propMap.EPCs(), func(device IPAndEOJ, success bool, properties Properties, failedEPCs []EPCType) (CallbackCompleteStatus, error) {
			defer close(callbackDone) // 必ず完了を通知

			h.registerProperties(device, properties)
			// デバイス情報を保存
			h.saveDeviceInfo()

			// 結果を記録
			fmt.Printf("デバイス %v のプロパティを %v個 更新しました\n", device, len(properties))
			return CallbackFinished, nil
		})

		if err != nil {
			errMutex.Lock()
			if firstErr == nil {
				firstErr = fmt.Errorf("プロパティ取得リクエストの送信に失敗: %w", err)
			}
			errMutex.Unlock()
			close(callbackDone) // エラー時も完了を通知
			wg.Done()
			continue
		}

		// 非同期でコールバックの完了またはタイムアウトを待つ
		go func(device IPAndEOJ) {
			defer wg.Done()
			select {
			case <-callbackDone:
				// コールバックが呼ばれた
			case <-timeoutCtx.Done():
				// タイムアウトした
				errMutex.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("デバイス %v のプロパティ更新がタイムアウトしました", device)
				}
				errMutex.Unlock()
			}
		}(device)
	}

	// 全てのデバイスの更新が完了するまで待つ
	wg.Wait()

	// エラーがあれば返す
	if firstErr != nil {
		return firstErr
	}

	return nil
}

func (h *ECHONETLiteHandler) SaveAliasFile() error {
	err := h.DeviceAliases.SaveToFile(DeviceAliasesFileName)
	if err != nil {
		return fmt.Errorf("エイリアス情報の保存に失敗しました: %w", err)
	}
	return nil
}

func (h *ECHONETLiteHandler) AliasList() []AliasDevicePair {
	return h.DeviceAliases.GetAllAliases()
}

func (h *ECHONETLiteHandler) GetAliases(device IPAndEOJ) []string {
	return h.DeviceAliases.GetAliases(device)
}

func (h *ECHONETLiteHandler) AliasSet(alias *string, criteria FilterCriteria) error {
	devices := h.devices.Filter(criteria)
	if devices.Len() == 0 {
		return fmt.Errorf("デバイスが見つかりません: %v", criteria)
	}
	if devices.Len() > 1 {
		return fmt.Errorf("デバイスが複数見つかりました: %v", devices)
	}
	found := devices.ListIPAndEOJ()[0]

	err := h.DeviceAliases.SetAlias(found, *alias)
	if err != nil {
		return fmt.Errorf("エイリアスを設定できませんでした: %w", err)
	}
	return h.SaveAliasFile()
}

func (h *ECHONETLiteHandler) AliasDelete(alias *string) error {
	if alias == nil {
		return errors.New("エイリアス名が指定されていません")
	}
	if err := h.DeviceAliases.RemoveAlias(*alias); err != nil {
		return fmt.Errorf("エイリアス %s の削除に失敗しました: %w", *alias, err)
	}
	return h.SaveAliasFile()
}

func (h *ECHONETLiteHandler) AliasGet(alias *string) (*IPAndEOJ, error) {
	if alias == nil {
		return nil, errors.New("エイリアス名が指定されていません")
	}
	device, ok := h.DeviceAliases.GetDeviceByAlias(*alias)
	if !ok {
		return nil, fmt.Errorf("エイリアス %s が見つかりません", *alias)
	}
	return &device, nil
}

// waitCallbackDone は、コールバック完了を待つ
func waitCallbackDone(ctx context.Context, executeOperation func(callbackDone chan struct{}) error, timeoutMessage string) error {
	// タイムアウト付きのコンテキストを作成（親コンテキストを使用）
	timeoutCtx, cancel := context.WithTimeout(ctx, CommandTimeout)
	defer cancel()

	// コールバック完了を待つためのチャネル
	callbackDone := make(chan struct{})

	// 操作の実行
	err := executeOperation(callbackDone)
	if err != nil {
		return fmt.Errorf("リクエスト送信エラー: %v", err)
	}

	// コールバックが呼ばれるか、タイムアウトするまで待つ
	select {
	case <-callbackDone:
		// コールバックが呼ばれた
	case <-timeoutCtx.Done():
		// コンテキストがキャンセルまたはタイムアウトした
		return errors.New(timeoutMessage)
	}

	return nil
}

func (h *ECHONETLiteHandler) GetDevices(deviceSpec *DeviceSpecifier) []IPAndEOJ {
	// フィルタリング条件を作成
	criteria := FilterCriteria{
		Device: deviceSpec,
	}

	// フィルタリング
	return h.devices.Filter(criteria).ListIPAndEOJ()
}
