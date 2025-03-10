package main

import (
	"context"
	"echonet-list/echonet_lite"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	DeviceFileName = "devices.json"
	CommandTimeout = 3 * time.Second // コマンド実行のタイムアウト時間
)

type DeviceClassNotFoundError struct {
	ClassCode echonet_lite.EOJClassCode
}

func (e DeviceClassNotFoundError) Error() string {
	return fmt.Sprintf("クラスコード %v のデバイスが見つかりません", e.ClassCode)
}

type TooManyDevicesError struct {
	ClassCode echonet_lite.EOJClassCode
	Devices   []DeviceInfo
}

func (e TooManyDevicesError) Error() string {
	errMsg := []string{
		fmt.Sprintf("クラスコード %v のデバイスが複数見つかりました。IPアドレスを指定してください", e.ClassCode),
	}
	for _, device := range e.Devices {
		errMsg = append(errMsg, fmt.Sprintf("  %s: %v", device.IP, device.EOJ))
	}
	return strings.Join(errMsg, "\n")
}

// ECHONETLiteHandler は、ECHONET Lite の通信処理を担当する構造体
type ECHONETLiteHandler struct {
	session      *Session
	devices      Devices
	localDevices DeviceProperties
	debug        bool
	ctx          context.Context    // コンテキスト
	cancel       context.CancelFunc // コンテキストのキャンセル関数
}

// SetDebug は、デバッグモードを設定する
func (h *ECHONETLiteHandler) SetDebug(debug bool) {
	h.debug = debug
	h.session.Debug = debug
	if logger != nil {
		logger.SetDebug(debug)
	}
}

// IsDebug は、現在のデバッグモードを返す
func (h *ECHONETLiteHandler) IsDebug() bool {
	return h.debug
}

// NewECHONETLiteHandler は、ECHONETLiteHandler の新しいインスタンスを作成する
func NewECHONETLiteHandler(ctx context.Context, ip net.IP, seoj echonet_lite.EOJ, debug bool) (*ECHONETLiteHandler, error) {
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

	localDevices := make(DeviceProperties)
	operationStatusOn := echonet_lite.OperationStatus(true)
	manufacturerCode := echonet_lite.ManufacturerCodeExperimental
	identificationNumber := echonet_lite.IdentificationNumber{
		ManufacturerCode: manufacturerCode,
		UniqueIdentifier: make([]byte, 13), // 識別番号未設定は13バイトの0
	}

	localDevices.Set(NodeProfileObject1,
		&operationStatusOn,
		&identificationNumber,
		&manufacturerCode,
		&echonet_lite.ECHONETLite_Version,
	)

	localDevices.Set(seoj,
		&operationStatusOn,
		&identificationNumber,
		&manufacturerCode,
	)

	// 最後にやること
	localDevices.UpdateProfileObjectProperties()

	handler := &ECHONETLiteHandler{
		session:      session,
		devices:      devices,
		localDevices: localDevices,
		debug:        debug,
		ctx:          handlerCtx,
		cancel:       cancel,
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
	EOJs := h.localDevices.GetInstanceList()
	return h.session.BroadcastNodeList(EOJs)
}

func (h *ECHONETLiteHandler) onReceiveMessage(ip net.IP, msg *echonet_lite.ECHONETLiteMessage) error {
	if msg == nil {
		return nil
	}

	if h.debug {
		fmt.Printf("%v: メッセージを受信: SEOJ:%v, DEOJ:%v, ESV:%v Property: %v\n",
			ip, msg.SEOJ, msg.DEOJ, msg.ESV,
			msg.Properties.String(msg.DEOJ.ClassCode()),
		)
	}

	eoj := msg.DEOJ
	_, ok := h.localDevices[eoj]
	if !ok {
		return fmt.Errorf("デバイス %v が見つかりません", eoj)
	}

	switch msg.ESV {
	case echonet_lite.ESVGet:
		responses, ok := h.localDevices.GetProperties(eoj, msg.Properties)

		ESV := echonet_lite.ESVGet_Res
		if !ok {
			ESV = echonet_lite.ESVGet_SNA
		}
		if h.debug {
			fmt.Printf("  Getメッセージに対する応答: %v\n", responses) // DEBUG
		}
		return h.session.SendResponse(ip, msg, ESV, responses, nil)

	case echonet_lite.ESVSetC, echonet_lite.ESVSetI:
		responses, success := h.localDevices.SetProperties(eoj, msg.Properties)

		if msg.ESV != echonet_lite.ESVSetI || !success {
			ESV := echonet_lite.ESVSetI_SNA
			if msg.ESV == echonet_lite.ESVSetC {
				if success {
					ESV = echonet_lite.ESVSet_Res
				} else {
					ESV = echonet_lite.ESVSetC_SNA
				}
			}
			if h.debug {
				fmt.Printf("  %vメッセージに対する応答: %v\n", msg.ESV, responses) // DEBUG
			}
			return h.session.SendResponse(ip, msg, ESV, responses, nil)
		}

	case echonet_lite.ESVSetGet:
		setResult, setSuccess := h.localDevices.SetProperties(eoj, msg.Properties)
		getResult, getSuccess := h.localDevices.GetProperties(eoj, msg.SetGetProperties)
		success := setSuccess && getSuccess

		ESV := echonet_lite.ESVSetGet_Res
		if !success {
			ESV = echonet_lite.ESVSetGet_SNA
		}
		if h.debug {
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
func (h *ECHONETLiteHandler) onInfMessage(ip net.IP, msg *echonet_lite.ECHONETLiteMessage) error {
	if msg == nil {
		if logger != nil {
			logger.Log("警告: 無効なINFメッセージを受信しました: nil")
		}
		return nil // 処理は継続
	}

	ipStr := ip.String()
	if logger != nil {
		logger.Log("INFメッセージを受信: %s, SEOJ:%v, DEOJ:%v", ipStr, msg.SEOJ, msg.DEOJ)
	}

	if !h.localDevices.IsAcceptableDEOJ(msg.DEOJ) {
		// 許容できないDEOJをもつmsgは破棄
		return nil
	}

	defer func() {
		if msg.ESV == echonet_lite.ESVINFC {
			replyProps := make([]echonet_lite.Property, 0, len(msg.Properties))
			// EDTをnilにする
			for _, p := range msg.Properties {
				replyProps = append(replyProps, echonet_lite.Property{
					EPC: p.EPC,
					EDT: nil,
				})
			}
			// 応答を返す
			err := h.session.SendResponse(ip, msg, echonet_lite.ESVINFC_Res, replyProps, nil)
			if err != nil {
				if logger != nil {
					logger.Log("エラー: INFメッセージに対する応答の送信に失敗: %v", err)
				}
			}
		}
	}()

	if msg.SEOJ.ClassCode() == echonet_lite.NodeProfile_ClassCode {
		// ノードプロファイルオブジェクトからのメッセージ
		for _, p := range msg.Properties {
			switch p.EPC {
			case echonet_lite.EPC_NPO_SelfNodeInstanceListS:
				err := h.onSelfNodeInstanceListS(ip, msg.SEOJ, true, p)
				if err != nil {
					if logger != nil {
						logger.Log("エラー: SelfNodeInstanceListSの処理中: %v", err)
					}
					return err
				}
			case echonet_lite.EPC_NPO_InstanceListNotification:
				iln := echonet_lite.DecodeInstanceListNotification(p.EDT)
				if iln == nil {
					if logger != nil {
						logger.Log("警告: InstanceListNotificationのデコードに失敗: %X", p.EDT)
					}
					return nil // 処理は継続
				}
				return h.onInstanceList(ip, msg.SEOJ, echonet_lite.InstanceList(*iln))
			default:
				if logger != nil {
					logger.Log("情報: 未処理のEPC: %v", p.EPC)
				}
			}
		}
	} else {
		// その他のオブジェクトからのメッセージ

		// IPアドレスが未登録の場合、デバイス情報を取得
		if !h.devices.HasIP(ipStr) {
			if logger != nil {
				logger.Log("情報: 未登録のIPアドレスからのメッセージ: %s", ipStr)
			}
			err := h.GetSelfNodeInstanceListS(ip)
			if err != nil {
				if logger != nil {
					logger.Log("エラー: SelfNodeInstanceListSの取得に失敗: %v", err)
				}
				return err
			}
		}

		// 未知のデバイスの場合、プロパティマップを取得
		if !h.devices.IsKnownDevice(ipStr, msg.SEOJ) {
			if logger != nil {
				logger.Log("情報: 新しいデバイスを検出: %s, %v", ipStr, msg.SEOJ)
			}
			fmt.Printf("%v: 新しいデバイスが検出されました: %v\n", ipStr, msg.SEOJ)
			err := h.GetGetPropertyMap(ip, msg.SEOJ)
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
			h.devices.RegisterProperties(ipStr, msg.SEOJ, msg.Properties)
			fmt.Printf("%v/%v: Propertyの通知: %v\n", ipStr, msg.SEOJ, msg.Properties)

			// デバイス情報を保存
			if err := h.devices.SaveToFile(DeviceFileName); err != nil {
				if logger != nil {
					logger.Log("警告: デバイス情報の保存に失敗しました: %v", err)
				}
				// 保存に失敗しても処理は継続
			}
		}
	}
	return nil
}

// onSelfNodeInstanceListS は、SelfNodeInstanceListSプロパティを受信したときのコールバック
func (h *ECHONETLiteHandler) onSelfNodeInstanceListS(ip net.IP, seoj echonet_lite.EOJ, success bool, p echonet_lite.Property) error {
	if !success {
		return fmt.Errorf("SelfNodeInstanceListSプロパティの取得に失敗しました: %v", ip)
	}

	if p.EPC != echonet_lite.EPC_NPO_SelfNodeInstanceListS {
		return fmt.Errorf("予期しないEPC: %v (期待値: %v)", p.EPC, echonet_lite.EPC_NPO_SelfNodeInstanceListS)
	}

	il := echonet_lite.DecodeSelfNodeInstanceListS(p.EDT)
	if il == nil {
		return fmt.Errorf("SelfNodeInstanceListSのデコードに失敗しました: %X", p.EDT)
	}
	return h.onInstanceList(ip, seoj, echonet_lite.InstanceList(*il))
}

func (h *ECHONETLiteHandler) onInstanceList(ip net.IP, seoj echonet_lite.EOJ, il echonet_lite.InstanceList) error {
	// デバイスの登録
	ipStr := ip.String()
	for _, eoj := range il {
		h.devices.RegisterDevice(ipStr, eoj)
		// ログ出力は削除（不要なため）
	}

	// デバイス情報の保存
	if err := h.devices.SaveToFile(DeviceFileName); err != nil {
		if logger != nil {
			logger.Log("警告: デバイス情報の保存に失敗しました: %v", err)
		}
		// 保存に失敗しても処理は継続
	}

	// 各デバイスのプロパティマップを取得
	var errors []error
	for _, eoj := range il {
		if err := h.GetGetPropertyMap(ip, eoj); err != nil {
			errors = append(errors, fmt.Errorf("デバイス %v のプロパティ取得に失敗: %w", eoj, err))
		}
	}

	// エラーがあれば報告（ただし処理は継続）
	if len(errors) > 0 {
		for _, err := range errors {
			if logger != nil {
				logger.Log("警告: %v", err)
			}
		}
	}

	return nil
}

// onGetPropertyMap は、GetPropertyMapプロパティを受信したときのコールバック
func (h *ECHONETLiteHandler) onGetPropertyMap(ip net.IP, seoj echonet_lite.EOJ, success bool, properties echonet_lite.Property) error {
	if !success {
		if logger != nil {
			logger.Log("警告: GetPropertyMapプロパティの取得に失敗しました: %v, %v", ip, seoj)
		}
		return CallbackFinished{} // 処理は継続
	}

	if properties.EPC != echonet_lite.EPCGetPropertyMap {
		if logger != nil {
			logger.Log("警告: 予期しないEPC: %v (期待値: %v)", properties.EPC, echonet_lite.EPCGetPropertyMap)
		}
		return CallbackFinished{} // 処理は継続
	}

	props := echonet_lite.DecodePropertyMap(properties.EDT)
	if props == nil {
		return echonet_lite.ErrInvalidPropertyMap{EDT: properties.EDT}
	}

	// 取得するプロパティのリストを作成
	forGet := make([]echonet_lite.EPCType, 0, len(props))
	for epc := range props {
		forGet = append(forGet, epc)
	}

	// プロパティが見つからない場合
	if len(forGet) == 0 {
		if logger != nil {
			logger.Log("情報: デバイス %v にプロパティが見つかりませんでした", seoj)
		}
		return CallbackFinished{}
	}

	// プロパティを取得
	err := h.session.GetProperties(ip,
		seoj,
		forGet,
		func(ip net.IP, seoj echonet_lite.EOJ, success bool, properties echonet_lite.Properties) error {
			if !success {
				// properties のうち、EDTが空の物を除去する
				// 失敗したプロパティも収集して表示する
				filteredProperties := make(echonet_lite.Properties, 0, len(properties))
				failedEPCs := make([]string, 0, len(properties))
				for _, p := range properties {
					if p.EDT != nil {
						filteredProperties = append(filteredProperties, p)
					} else {
						failedEPCs = append(failedEPCs, p.EPC.StringForClass(seoj.ClassCode()))
					}
				}
				properties = filteredProperties
				if logger != nil {
					logger.Log("警告: プロパティ取得に失敗しました: %v, %v, Failed EPCs: %v", ip, seoj, failedEPCs)
				}
			}

			// プロパティを登録
			h.devices.RegisterProperties(ip.String(), seoj, properties)

			// デバイス情報を保存
			if err := h.devices.SaveToFile(DeviceFileName); err != nil {
				if logger != nil {
					logger.Log("警告: デバイス情報の保存に失敗しました: %v", err)
				}
				// 保存に失敗しても処理は継続
			}

			return CallbackFinished{}
		},
	)

	if err != nil {
		if logger != nil {
			logger.Log("エラー: プロパティ取得リクエストの送信に失敗: %v", err)
		}
	}

	return CallbackFinished{}
}

// GetSelfNodeInstanceListS は、SelfNodeInstanceListSプロパティを取得する
func (h *ECHONETLiteHandler) GetSelfNodeInstanceListS(ip net.IP) error {
	return h.session.GetSelfNodeInstanceListS(ip, h.onSelfNodeInstanceListS)
}

// GetGetPropertyMap は、GetPropertyMapプロパティを取得する
func (h *ECHONETLiteHandler) GetGetPropertyMap(ip net.IP, eoj echonet_lite.EOJ) error {
	return h.session.GetProperty(ip, eoj, echonet_lite.EPCGetPropertyMap, h.onGetPropertyMap)
}

// Discover は、ECHONET Liteデバイスを検出する
func (h *ECHONETLiteHandler) Discover() error {
	return h.GetSelfNodeInstanceListS(BroadcastIP)
}

// ListDevices は、検出されたデバイスの一覧を表示する
func (h *ECHONETLiteHandler) ListDevices(criteria FilterCriteria, propMode PropertyMode) string {
	// フィルタリングを実行
	filtered := h.devices.Filter(criteria)

	// EPCが指定された場合は常にそのプロパティを表示
	if len(criteria.EPCs) > 0 {
		propMode = PropAll
	}

	// 文字列化して返す
	return filtered.StringWithPropertyMode(propMode)
}

// validateEPCsInPropertyMap は、指定されたEPCがプロパティマップに含まれているかを確認する
func (h *ECHONETLiteHandler) validateEPCsInPropertyMap(ip string, eoj echonet_lite.EOJ, epcs []echonet_lite.EPCType, mapType PropertyMapType) (bool, []echonet_lite.EPCType, error) {
	invalidEPCs := []echonet_lite.EPCType{}

	// デバイスが存在するか確認
	if !h.devices.IsKnownDevice(ip, eoj) {
		return false, invalidEPCs, fmt.Errorf("デバイスが見つかりません: %s, %v", ip, eoj)
	}

	// 各EPCがプロパティマップに含まれているか確認
	for _, epc := range epcs {
		if !h.devices.HasEPCInPropertyMap(ip, eoj, mapType, epc) {
			invalidEPCs = append(invalidEPCs, epc)
		}
	}

	return len(invalidEPCs) == 0, invalidEPCs, nil
}

// GetProperties は、プロパティ値を取得する
// 成功時には ip, eoj と properties を返す
func (h *ECHONETLiteHandler) GetProperties(cmd *Command) (net.IP, echonet_lite.EOJ, echonet_lite.Properties, error) {
	// 結果を格納する変数
	var resultIP net.IP
	var resultEOJ echonet_lite.EOJ
	var resultProperties echonet_lite.Properties

	err := handlePropertyCommand(h.ctx, cmd, h.devices, func(targetIP string) (bool, error) {
		if cmd.GetClassCode() == nil || cmd.GetInstanceCode() == nil {
			return false, errors.New("get コマンドにはクラスコード、インスタンスコードが必要です")
		}
		if len(cmd.EPCs) == 0 {
			return false, errors.New("get コマンドには少なくとも1つのEPCが必要です")
		}

		// 指定されたEPCがGetPropertyMapに含まれているか確認
		valid, invalidEPCs, err := h.validateEPCsInPropertyMap(targetIP, echonet_lite.MakeEOJ(*cmd.GetClassCode(), *cmd.GetInstanceCode()), cmd.EPCs, GetPropertyMap)
		if err != nil {
			return false, err
		}
		if !valid {
			return false, fmt.Errorf("以下のEPCはGetPropertyMapに含まれていません: %v", invalidEPCs)
		}

		return true, nil
	}, func(ip net.IP, deoj echonet_lite.EOJ, callbackDone chan struct{}) error {
		// コンテキストは親コンテキスト（h.ctx）を使用
		err := h.session.GetProperties(ip, deoj, cmd.EPCs, func(ip net.IP, eoj echonet_lite.EOJ, success bool, properties echonet_lite.Properties) error {
			defer close(callbackDone) // 必ず完了を通知

			if !success {
				errMsg := fmt.Sprintf("プロパティ取得失敗: %s, %v", ip, eoj)
				if logger != nil {
					logger.Log("エラー: %s", errMsg)
				}
				cmd.Error = errors.New(errMsg)
				return CallbackFinished{}
			}

			// 成功した場合の処理
			resultIP = ip
			resultEOJ = eoj
			resultProperties = properties

			// プロパティの登録
			h.devices.RegisterProperties(ip.String(), eoj, properties)

			// デバイス情報を保存
			if err := h.devices.SaveToFile(DeviceFileName); err != nil {
				if logger != nil {
					logger.Log("警告: デバイス情報の保存に失敗しました: %v", err)
				}
				// 保存に失敗しても処理は継続
			}

			return CallbackFinished{}
		})

		if err != nil {
			if logger != nil {
				logger.Log("エラー: プロパティ取得リクエストの送信に失敗: %v", err)
			}
			cmd.Error = fmt.Errorf("プロパティ取得リクエストの送信に失敗: %w", err)
			close(callbackDone) // エラー時も完了を通知
		}

		return err
	}, "プロパティ取得がタイムアウトしました")
	return resultIP, resultEOJ, resultProperties, err
}

// SetProperties は、プロパティ値を設定する
func (h *ECHONETLiteHandler) SetProperties(cmd *Command) (net.IP, echonet_lite.EOJ, echonet_lite.Properties, error) {
	// 結果を格納する変数
	var resultIP net.IP
	var resultEOJ echonet_lite.EOJ
	var resultProperties echonet_lite.Properties

	err := handlePropertyCommand(h.ctx, cmd, h.devices, func(targetIP string) (bool, error) {
		if cmd.GetClassCode() == nil || cmd.GetInstanceCode() == nil {
			return false, errors.New("set コマンドにはクラスコード、インスタンスコードが必要です")
		}
		if len(cmd.Properties) == 0 {
			return false, errors.New("set コマンドには少なくとも1つのプロパティが必要です")
		}

		// 指定されたEPCがSetPropertyMapに含まれているか確認
		// Propertiesから各EPCを抽出
		epcs := make([]echonet_lite.EPCType, 0, len(cmd.Properties))
		for _, prop := range cmd.Properties {
			epcs = append(epcs, prop.EPC)
		}

		valid, invalidEPCs, err := h.validateEPCsInPropertyMap(targetIP, echonet_lite.MakeEOJ(*cmd.GetClassCode(), *cmd.GetInstanceCode()), epcs, SetPropertyMap)
		if err != nil {
			return false, err
		}
		if !valid {
			return false, fmt.Errorf("以下のEPCはSetPropertyMapに含まれていません: %v", invalidEPCs)
		}

		return true, nil
	}, func(ip net.IP, deoj echonet_lite.EOJ, callbackDone chan struct{}) error {
		// コンテキストは親コンテキスト（h.ctx）を使用
		err := h.session.SetProperties(ip, deoj, cmd.Properties, func(ip net.IP, eoj echonet_lite.EOJ, success bool, properties echonet_lite.Properties) error {
			defer close(callbackDone) // 必ず完了を通知

			if !success {
				errMsg := fmt.Sprintf("プロパティ設定失敗: %s, %v", ip, eoj)
				if logger != nil {
					logger.Log("エラー: %s", errMsg)
				}
				cmd.Error = errors.New(errMsg)
				return CallbackFinished{}
			}

			// 成功した場合の処理
			resultIP = ip
			resultEOJ = eoj
			resultProperties = cmd.Properties

			// 戻ってきた properties ではなく、こちらが設定した cmd.Properties で登録する
			h.devices.RegisterProperties(ip.String(), eoj, cmd.Properties)

			// デバイス情報を保存
			if err := h.devices.SaveToFile(DeviceFileName); err != nil {
				if logger != nil {
					logger.Log("警告: デバイス情報の保存に失敗しました: %v", err)
				}
				// 保存に失敗しても処理は継続
			}

			return CallbackFinished{}
		})

		if err != nil {
			if logger != nil {
				logger.Log("エラー: プロパティ設定リクエストの送信に失敗: %v", err)
			}
			cmd.Error = fmt.Errorf("プロパティ設定リクエストの送信に失敗: %w", err)
			close(callbackDone) // エラー時も完了を通知
		}

		return err
	}, "プロパティ設定がタイムアウトしました")
	return resultIP, resultEOJ, resultProperties, err
}

// UpdateProperties は、フィルタリングされたデバイスのプロパティキャッシュを更新する
func (h *ECHONETLiteHandler) UpdateProperties(cmd *Command) error {
	// フィルタリング条件を作成
	criteria := FilterCriteria{
		IPAddress:    cmd.GetIPAddress(),
		ClassCode:    cmd.GetClassCode(),
		InstanceCode: cmd.GetInstanceCode(),
	}

	// フィルタリングを実行
	filtered := h.devices.Filter(criteria)

	// フィルタリング結果が空の場合
	if len(filtered.data) == 0 {
		return fmt.Errorf("条件に一致するデバイスが見つかりません")
	}

	fmt.Printf("更新対象のデバイス:\n%s\n", filtered.String())
	fmt.Println("プロパティの更新を開始します...")

	// タイムアウト付きのコンテキストを作成（親コンテキストを使用）
	timeoutCtx, cancel := context.WithTimeout(h.ctx, CommandTimeout)
	defer cancel()

	// 全てのデバイスの更新完了を待つためのWaitGroup
	var wg sync.WaitGroup
	var errMutex sync.Mutex
	var firstErr error

	// 各デバイスに対して処理を実行
	for ip, eojMap := range filtered.data {
		for eoj := range eojMap {
			wg.Add(1)

			ipAddr := net.ParseIP(ip)
			if ipAddr == nil {
				errMutex.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("無効なIPアドレス: %s", ip)
				}
				errMutex.Unlock()
				wg.Done()
				continue
			}

			propMap := h.devices.GetPropertyMap(ip, eoj, GetPropertyMap)
			if propMap == nil {
				errMutex.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("プロパティマップが見つかりません: %s, %v", ip, eoj)
				}
				errMutex.Unlock()
				wg.Done()
				continue
			}

			// コールバック完了を待つためのチャネル
			callbackDone := make(chan struct{})

			err := h.session.GetProperties(ipAddr, eoj, propMap.EPCs(), func(ipAddress net.IP, eoj echonet_lite.EOJ, success bool, properties echonet_lite.Properties) error {
				defer close(callbackDone) // 必ず完了を通知

				if !success {
					nonEmpties := make(echonet_lite.Properties, 0, len(properties))
					for _, p := range properties {
						if p.EDT != nil {
							nonEmpties = append(nonEmpties, p)
						}
					}
					properties = nonEmpties
				}
				h.devices.RegisterProperties(ipAddress.String(), eoj, properties)
				// デバイス情報を保存
				if err := h.devices.SaveToFile(DeviceFileName); err != nil {
					if logger != nil {
						logger.Log("警告: デバイス情報の保存に失敗しました: %v", err)
					}
					// 保存に失敗しても処理は継続
				}

				// 結果を記録
				fmt.Printf("デバイス %s, %s のプロパティを %v個 更新しました\n", ipAddress, eoj, len(properties))
				return CallbackFinished{}
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
			go func(ip string, eoj echonet_lite.EOJ) {
				defer wg.Done()
				select {
				case <-callbackDone:
					// コールバックが呼ばれた
				case <-timeoutCtx.Done():
					// タイムアウトした
					errMutex.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("デバイス %s, %s のプロパティ更新がタイムアウトしました", ip, eoj)
					}
					errMutex.Unlock()
				}
			}(ip, eoj)
		}
	}

	// 全てのデバイスの更新が完了するまで待つ
	wg.Wait()

	// エラーがあれば返す
	if firstErr != nil {
		return firstErr
	}

	return nil
}

// プロパティ操作コマンド（get/set）の共通処理を行うヘルパー関数
func handlePropertyCommand(ctx context.Context, cmd *Command, devices Devices, requiredParamCheck func(targetIP string) (bool, error), executeOperation func(ip net.IP, deoj echonet_lite.EOJ, callbackDone chan struct{}) error, timeoutMessage string) error {
	// 必要なパラメータのチェック
	// IPアドレスが指定されていない場合、クラスコードに一致するデバイスを探す
	var targetIP string
	if cmd.GetIPAddress() == nil {
		// クラスコードとインスタンスコードに一致するデバイスを検索
		matchingDevices := devices.FindDevicesByClassAndInstance(cmd.GetClassCode(), cmd.GetInstanceCode())

		if len(matchingDevices) == 0 {
			cmd.Error = DeviceClassNotFoundError{ClassCode: *cmd.GetClassCode()}
			return cmd.Error
		} else if len(matchingDevices) > 1 {
			cmd.Error = TooManyDevicesError{ClassCode: *cmd.GetClassCode(), Devices: matchingDevices}
			return cmd.Error
		}

		// 一致するデバイスが1つだけの場合、そのIPアドレスを使用
		targetIP = matchingDevices[0].IP
	} else {
		targetIP = *cmd.GetIPAddress()
	}

	if ok, err := requiredParamCheck(targetIP); !ok {
		cmd.Error = err
		return cmd.Error
	}

	// 宛先アドレスの作成
	ip := net.ParseIP(targetIP)
	if ip == nil {
		cmd.Error = fmt.Errorf("無効なIPアドレス: %v", targetIP)
		return cmd.Error
	}

	var err error

	// 宛先EOJの作成
	deoj := echonet_lite.MakeEOJ(*cmd.GetClassCode(), *cmd.GetInstanceCode())

	// タイムアウト付きのコンテキストを作成（親コンテキストを使用）
	timeoutCtx, cancel := context.WithTimeout(ctx, CommandTimeout)
	defer cancel()

	// コールバック完了を待つためのチャネル
	callbackDone := make(chan struct{})

	// 操作の実行
	err = executeOperation(ip, deoj, callbackDone)
	if err != nil {
		cmd.Error = fmt.Errorf("リクエスト送信エラー: %v", err)
		return cmd.Error
	}

	// コールバックが呼ばれるか、タイムアウトするまで待つ
	select {
	case <-callbackDone:
		// コールバックが呼ばれた
	case <-timeoutCtx.Done():
		// コンテキストがキャンセルまたはタイムアウトした
		cmd.Error = errors.New(timeoutMessage)
	}

	return cmd.Error
}
