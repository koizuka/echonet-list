package echonet_lite

import (
	"context"
	"echonet-list/echonet_lite/log"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

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

// onInfMessage は、INFメッセージを受信したときのコールバック
func (h *ECHONETLiteHandler) onInfMessage(ip net.IP, msg *ECHONETLiteMessage) error {
	logger := log.GetLogger()
	if msg == nil {
		if logger != nil {
			logger.Log("警告: 無効なINFメッセージを受信しました: nil")
		}
		return nil // 処理は継続
	}

	if logger != nil {
		logger.Log("INFメッセージを受信: %v, SEOJ:%v, DEOJ:%v", ip, msg.SEOJ, msg.DEOJ)
	}
	fmt.Printf("INFメッセージを受信: %v %v, DEOJ:%v\n", ip, msg.SEOJ, msg.DEOJ) // DEBUG

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
			if err != nil && logger != nil {
				logger.Log("エラー: INFメッセージに対する応答の送信に失敗: %v", err)
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
	h.propMutex.Lock()
	defer h.propMutex.Unlock()

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
			if logger := log.GetLogger(); logger != nil {
				logger.Log("警告: %v", err)
			}
		}
	}

	return nil
}

// onGetPropertyMap は、GetPropertyMapプロパティを受信したときのコールバック
func (h *ECHONETLiteHandler) onGetPropertyMap(device IPAndEOJ, success bool, properties Properties, _ []EPCType) (CallbackCompleteStatus, error) {
	logger := log.GetLogger()
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
	err := h.session.StartGetPropertiesWithRetry(
		h.ctx,
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

			// failedEPCs は GetPropertyMap から取り除くことで、次回以降取得しないようにする
			for _, epc := range failedEPCs {
				props.Delete(epc)
			}
			h.registerProperties(device, []Property{{
				EPC: EPCGetPropertyMap,
				EDT: props.Encode(),
			}})

			// デバイス情報を保存
			h.saveDeviceInfo()

			return CallbackFinished, nil
		},
	)

	if err != nil && logger != nil {
		logger.Log("エラー: プロパティ取得リクエストの送信に失敗: %v", err)
	}

	return CallbackFinished, err
}

// GetSelfNodeInstanceListS は、SelfNodeInstanceListSプロパティを取得する
func (h *ECHONETLiteHandler) GetSelfNodeInstanceListS(ip net.IP) error {
	isBroadcast := ip.Equal(BroadcastIP)
	// broadcastの場合、1秒無通信で完了とする
	// タイマーを作る
	var timer *time.Timer
	idleTimeout := time.Duration(2 * time.Second)
	if isBroadcast {
		timer = time.NewTimer(idleTimeout)
		defer timer.Stop()
	}
	key, err := h.session.StartGetProperties(
		IPAndEOJ{ip, NodeProfileObject}, []EPCType{EPC_NPO_SelfNodeInstanceListS},
		func(ie IPAndEOJ, b bool, p Properties, f []EPCType) (CallbackCompleteStatus, error) {
			var completeStatus CallbackCompleteStatus
			if isBroadcast {
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
	if isBroadcast {
		defer h.session.unregisterCallback(key)

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
func (h *ECHONETLiteHandler) GetGetPropertyMap(device IPAndEOJ) error {
	return h.session.StartGetPropertiesWithRetry(h.ctx, device, []EPCType{EPCGetPropertyMap}, h.onGetPropertyMap)
}

// Discover は、ECHONET Liteデバイスを検出する
func (h *ECHONETLiteHandler) Discover() error {
	return h.GetSelfNodeInstanceListS(BroadcastIP)
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

// GetProperties は、プロパティ値を取得する
// 成功時には ip, eoj と properties を返す
func (h *ECHONETLiteHandler) GetProperties(device IPAndEOJ, EPCs []EPCType, skipValidation bool) (DeviceAndProperties, error) {
	// 結果を格納する変数
	var result DeviceAndProperties

	logger := log.GetLogger()

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
		if logger != nil {
			logger.Log("エラー: プロパティ取得に失敗: %v", err)
		}
		return DeviceAndProperties{}, fmt.Errorf("%v: プロパティ取得に失敗: %w", device, err)
	}

	// 成功したプロパティを登録（部分的な成功の場合も含む）
	if len(properties) > 0 {
		// プロパティの登録
		h.registerProperties(device, properties)

		// デバイス情報を保存
		h.saveDeviceInfo()
	}

	// 結果を設定
	result.Device = device
	result.Properties = properties

	// 全体の成功/失敗を判定
	if !success {
		errMsg := fmt.Sprintf("%v: 一部のプロパティ取得に失敗: %v", device, failedEPCs)
		if logger != nil {
			logger.Log("警告: %s", errMsg)
		}
		return result, errors.New(errMsg)
	}

	return result, nil
}

// SetProperties は、プロパティ値を設定する
func (h *ECHONETLiteHandler) SetProperties(device IPAndEOJ, properties Properties) (DeviceAndProperties, error) {
	// 結果を格納する変数
	var result DeviceAndProperties

	logger := log.GetLogger()

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
		if logger != nil {
			logger.Log("エラー: プロパティ設定に失敗: %v", err)
		}
		return DeviceAndProperties{}, fmt.Errorf("プロパティ設定に失敗: %w", err)
	}

	// 成功したプロパティを登録（部分的な成功の場合も含む）
	if len(successProperties) > 0 {
		// プロパティの登録
		h.registerProperties(device, successProperties)

		// デバイス情報を保存
		h.saveDeviceInfo()
	}

	// 結果を設定
	result.Device = device
	result.Properties = successProperties

	// 全体の成功/失敗を判定
	if !success {
		errMsg := fmt.Sprintf("一部のプロパティ設定に失敗: %v: %v", device, failedEPCs)
		if logger != nil {
			logger.Log("警告: %s", errMsg)
		}
		return result, errors.New(errMsg)
	}

	return result, nil
}

// UpdateProperties は、フィルタリングされたデバイスのプロパティキャッシュを更新する
// force が true の場合、最終更新時刻に関わらず強制的に更新する
func (h *ECHONETLiteHandler) UpdateProperties(criteria FilterCriteria, force bool) error {
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
	storeError := func(err error) {
		errMutex.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMutex.Unlock()
	}

	// 各デバイスに対して処理を実行
	for _, device := range filtered.ListIPAndEOJ() {
		// forceがfalseの場合、最終更新時刻をチェック
		if !force {
			lastUpdateTime := h.devices.GetLastUpdateTime(device)
			if !lastUpdateTime.IsZero() && time.Since(lastUpdateTime) < UpdateIntervalThreshold {
				fmt.Printf("デバイス %v は最近更新されたためスキップします (最終更新: %v)\n", device, lastUpdateTime.Format(time.RFC3339))
				continue // 更新をスキップ
			}
		}

		wg.Add(1)

		propMap := h.devices.GetPropertyMap(device, GetPropertyMap)
		if propMap == nil {
			storeError(fmt.Errorf("プロパティマップが見つかりません: %v", device))
			wg.Done()
			continue
		}

		// 各デバイスに対して並列処理を実行
		go func(device IPAndEOJ, propMap PropertyMap) {
			defer wg.Done()

			success, properties, failedEPCs, err := h.session.GetProperties(
				timeoutCtx,
				device,
				propMap.EPCs(),
			)

			if err != nil {
				storeError(fmt.Errorf("デバイス %v のプロパティ取得に失敗: %w", device, err))
				return
			}

			// 成功したプロパティを登録（部分的な成功の場合も含む）
			if len(properties) > 0 {
				h.registerProperties(device, properties)
				// デバイス情報を保存
				h.saveDeviceInfo()
			}

			// 結果を記録
			fmt.Printf("デバイス %v のプロパティを %v個 更新しました\n", device, len(properties))

			// 全体の成功/失敗を判定
			if !success && len(failedEPCs) > 0 {
				storeError(fmt.Errorf("デバイス %v の一部のプロパティ取得に失敗: %v", device, failedEPCs))
			}
		}(device, propMap)
	}

	// 全てのデバイスの更新が完了するまで待つ
	wg.Wait()

	// エラーがあれば返す
	if firstErr != nil {
		return firstErr
	}

	return nil
}
