package main

import (
	"fmt"
	"net"
	"time"
)

// ECHONET Lite 資料
// https://echonet.jp/wp/wp-content/uploads/pdf/General/Standard/Echonet_lite_old/Echonet_lite_V1_11_jp/ECHONET-Lite_Ver.1.11_02.pdf

// ECHONETLiteMessage はECHONET Liteのメッセージを表します。
type ECHONETLiteMessage struct {
	EHD        []byte     // ヘッダ（2バイト）
	TID        []byte     // トランザクションID（2バイト）
	SEOJ       []byte     // 送信元ECHONETオブジェクト（3バイト）
	DEOJ       []byte     // 宛先ECHONETオブジェクト（3バイト）
	ESV        byte       // サービスコード（1バイト）
	OPC        byte       // プロパティ数（1バイト）
	Properties []Property // プロパティリスト
}

// Property は各プロパティ（EPC, PDC, EDT）を表します。
type Property struct {
	EPC byte   // プロパティコード
	PDC byte   // EDTのバイト数
	EDT []byte // プロパティデータ
}

// parseECHONETLiteMessage は受信したバイト列からECHONET Liteメッセージをパースします。
func parseECHONETLiteMessage(data []byte) (*ECHONETLiteMessage, error) {
	// 最低限、EHD(2)+TID(2)+SEOJ(3)+DEOJ(3)+ESV(1)+OPC(1)=12バイトは必要
	if len(data) < 12 {
		return nil, fmt.Errorf("パケットが短すぎます: %d バイト", len(data))
	}

	msg := &ECHONETLiteMessage{
		EHD:  data[0:2],
		TID:  data[2:4],
		SEOJ: data[4:7],
		DEOJ: data[7:10],
		ESV:  data[10],
		OPC:  data[11],
	}
	pos := 12
	msg.Properties = make([]Property, 0, msg.OPC)
	for i := 0; i < int(msg.OPC); i++ {
		if pos+2 > len(data) {
			return nil, fmt.Errorf("プロパティの長さが不正です")
		}
		prop := Property{
			EPC: data[pos],
			PDC: data[pos+1],
		}
		pos += 2
		if int(prop.PDC) > 0 {
			if pos+int(prop.PDC) > len(data) {
				return nil, fmt.Errorf("EDTの長さが不正です")
			}
			prop.EDT = data[pos : pos+int(prop.PDC)]
			pos += int(prop.PDC)
		}
		msg.Properties = append(msg.Properties, prop)
	}
	return msg, nil
}

type UDPConnection struct {
	RemoteAddr  *net.UDPAddr
	LocalAddr   *net.UDPAddr
	SendConn    *net.UDPConn
	ReceiveConn *net.UDPConn
}

func CreateUDPConnection(remoteAddr string) (*UDPConnection, error) {
	remote, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("リモートアドレスの解決に失敗: %w", err)
	}

	sendConn, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		return nil, fmt.Errorf("UDP接続に失敗: %w", err)
	}

	local := sendConn.LocalAddr().(*net.UDPAddr)

	/*
		receiveConn := sendConn
		/*/
	receiveConn, err := net.ListenUDP("udp", local)
	if err != nil {
		_ = sendConn.Close()
		return nil, fmt.Errorf("UDP受信エラー: %w", err)
	}
	// */

	conn := &UDPConnection{
		LocalAddr:   local,
		RemoteAddr:  remote,
		SendConn:    sendConn,
		ReceiveConn: receiveConn,
	}
	return conn, nil
}

func (c *UDPConnection) Close() {
	c.SendConn.Close()
	if c.ReceiveConn != c.SendConn {
		c.ReceiveConn.Close()
	}
}

func (c *UDPConnection) Send(data []byte) (int, error) {
	return c.SendConn.Write(data)
}

func (c *UDPConnection) Receive() ([]byte, *net.UDPAddr, error) {
	buf := make([]byte, 1024)
	n, addr, err := c.ReceiveConn.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, err
	}
	return buf[:n], addr, nil
}

func main() {
	// local address （ポート3610はECHONET Liteの既定ポート）
	/*
		local, err := net.ResolveUDPAddr("udp", ":3610")
		if err != nil {
			fmt.Println("UDP受信エラー:", err)
			return
		}
	*/

	// broadcastAddr := "255.255.255.255:3610"
	broadcastAddr := "192.168.0.255:3610"
	// broadcastAddr := "[ff02::1]:3610"
	// broadcastAddr := "192.168.0.212:3610" // リンクプラス無線アダプタのIPアドレス
	conn, err := CreateUDPConnection(broadcastAddr)
	if err != nil {
		fmt.Println("接続に失敗:", err)
		return
	}
	defer conn.Close()

	// --- ECHONET Lite 探索要求パケットの作成 ---
	// パケット構造：
	// 1. EHD (2バイト): 0x10 0x81 （ECHONET Liteの固定ヘッダ）
	// 2. TID (2バイト): 任意のトランザクションID（ここでは 0x00 0x01）
	// 3. SEOJ (3バイト): 送信元オブジェクト（ここではコントローラー例 0x05, 0xFF, 0x01）
	// 4. DEOJ (3バイト): 宛先オブジェクト（ノードプロフィールオブジェクト例 0x0E, 0xF0, 0x01）
	// 5. ESV (1バイト): サービスコード（Get要求 0x62）
	// 6. OPC (1バイト): プロパティ数（1）
	// 7. プロパティ:
	//      EPC (1バイト): 0x80（例として、動作状態などを問い合わせる）
	//      PDC (1バイト): 0x00（データなし）
	packet := []byte{
		0x10, 0x81, // EHD
		0x00, 0x01, // TID
		0x05, 0xFF, 0x01, // SEOJ（送信元：コントローラー例）
		0x0E, 0xF0, 0x01, // DEOJ（宛先：ノードプロフィールオブジェクト）
		0x62,       // ESV: Get要求
		0x01,       // OPC: 1プロパティ
		0x80, 0x00, // プロパティ: EPC 0x80, PDC 0（データなし）
	}

	// 探索要求パケットを送信
	_, err = conn.Send(packet)
	if err != nil {
		fmt.Println("パケット送信に失敗:", err)
		return
	}
	fmt.Println("探索要求パケットを送信しました。応答を待ちます...")

	// 応答受信用にタイムアウトを設定（ここでは5秒間待つ）
	if err := conn.ReceiveConn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		fmt.Println("読み込みタイムアウト設定に失敗:", err)
		return
	}

	// 応答を受信し、各機器の情報を記録
	devices := make(map[string]*ECHONETLiteMessage)
	for {
		data, addr, err := conn.Receive()
		if err != nil {
			// タイムアウトであれば受信終了
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			fmt.Println("データ受信エラー:", err)
			break
		}
		msg, err := parseECHONETLiteMessage(data)
		if err != nil {
			fmt.Println("パケット解析エラー:", err)
			continue
		}
		fmt.Printf("応答を受信: %s から --- SEOJ: % X, DEOJ: % X, ESV: % X, OPC: %d\n",
			addr.IP, msg.SEOJ, msg.DEOJ, msg.ESV, msg.OPC)
		devices[addr.IP.String()] = msg
	}

	// 検出されたECHONET Lite機器一覧を表示
	fmt.Println("\n検出されたECHONET Lite機器一覧:")
	for ip, msg := range devices {
		fmt.Printf("IP: %s, 送信元(SEOJ): % X, 宛先(DEOJ): % X, サービス(ESV): % X\n",
			ip, msg.SEOJ, msg.DEOJ, msg.ESV)
	}
}
