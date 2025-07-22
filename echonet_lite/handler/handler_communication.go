package handler

import (
	"context"
	"echonet-list/echonet_lite"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

// CommunicationHandler は、ECHONET Lite 通信機能を担当する構造体
type CommunicationHandler struct {
	session      *Session          // セッション
	localDevices DeviceProperties  // 自ノードが所有するデバイスのプロパティ
	dataAccessor DataAccessor      // データアクセス機能
	notifier     NotificationRelay // 通知中継
	ctx          context.Context   // コンテキスト
	Debug        bool              // デバッグモード
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
	return &CommunicationHandler{
		session:      session,
		localDevices: localDevices,
		dataAccessor: dataAccessor,
		notifier:     notifier,
		ctx:          ctx,
		Debug:        debug,
	}
}

// SetDebug は、デバッグモードを設定する
func (h *CommunicationHandler) SetDebug(debug bool) {
	h.Debug = debug
	h.session.Debug = debug
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

	// デバイスの登録
	for _, eoj := range il {
		h.dataAccessor.RegisterDevice(IPAndEOJ{IP: ip, EOJ: eoj})
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
		errMsg := fmt.Sprintf("%v: 一部のプロパティ取得に失敗: %v", device, failedEPCs)
		slog.Warn("警告", "msg", errMsg)
		return result, errors.New(errMsg)
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
		errMsg := fmt.Sprintf("一部のプロパティ設定に失敗: %v: %v", device, failedEPCs)
		slog.Warn("警告", "device", device, "msg", errMsg)
		return result, errors.New(errMsg)
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

	slog.Info("Filtered devices for update", "device_count", filtered.Len(), "criteria", criteria)

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

	sameIPDelay := 0
	lastIP := net.IP{}
	SameIPDelayDuration := 100 * time.Millisecond

	// 各デバイスに対して処理を実行
	for _, device := range filtered.ListIPAndEOJ() {
		// forceがfalseの場合、最終更新時刻をチェック
		if !force {
			lastUpdateTime := h.dataAccessor.GetLastUpdateTime(device)
			if !lastUpdateTime.IsZero() && time.Since(lastUpdateTime) < UpdateIntervalThreshold {
				// fmt.Printf("デバイス %v は最近更新されたためスキップします (最終更新: %v)\n", device, lastUpdateTime.Format(time.RFC3339))
				continue // 更新をスキップ
			}
			if h.dataAccessor.IsOffline(device) {
				continue // オフラインのデバイスはスキップ
			}
		}

		wg.Add(1)

		propMap := h.dataAccessor.GetPropertyMap(device, GetPropertyMap)
		if propMap == nil {
			storeError(fmt.Errorf("プロパティマップが見つかりません: %v", device))
			wg.Done()
			continue
		}

		// 同じIPアドレスのデバイスに対しては、遅延を追加(床暖房が連続送信していると再送が発生しているため)
		if device.IP.Equal(lastIP) {
			sameIPDelay++
		} else {
			sameIPDelay = 0
			lastIP = device.IP
		}

		// 各デバイスに対して並列処理を実行
		go func(device IPAndEOJ, propMap PropertyMap, delay time.Duration) {
			defer wg.Done()
			deviceName := h.dataAccessor.DeviceStringWithAlias(device)

			// 同じIPアドレスのデバイスに対して遅延を追加
			if delay > 0 {
				time.Sleep(delay)
			}

			success, properties, failedEPCs, err := h.session.GetProperties(
				h.ctx,
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
				storeError(fmt.Errorf("%v の一部のプロパティ取得に失敗: %v", deviceName, epcNames))
			}
		}(device, propMap, time.Duration(sameIPDelay)*SameIPDelayDuration)
	}

	// 全てのデバイスの更新が完了するまで待つ
	slog.Debug("Waiting for all device updates to complete")
	wg.Wait()

	duration := time.Since(start)
	// エラーがあれば返す
	if firstErr != nil {
		slog.Error("UpdateProperties completed with errors", "duration", duration, "first_error", firstErr)
		return firstErr
	}

	slog.Info("UpdateProperties completed successfully", "duration", duration)
	return nil
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
