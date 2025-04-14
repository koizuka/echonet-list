package console

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

const historyFileName = ".echonet_history"

// getHistoryFilePath は履歴ファイルのパスを取得する
func getHistoryFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// ホームディレクトリが取得できない場合はカレントディレクトリに作成
		log.Printf("警告: ホームディレクトリが取得できませんでした。履歴ファイルはカレントディレクトリに作成されます: %v", err)
		return historyFileName
	}
	return fmt.Sprintf("%s/%s", home, historyFileName)
}

// loadHistory は履歴ファイルから履歴を読み込む
func loadHistory(filePath string) []string {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{} // ファイルが存在しない場合は空の履歴
		}
		log.Printf("警告: 履歴ファイルの読み込みに失敗しました (%s): %v", filePath, err)
		return []string{}
	}
	defer file.Close()

	var history []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		history = append(history, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Printf("警告: 履歴ファイルのスキャン中にエラーが発生しました (%s): %v", filePath, err)
	}
	// 履歴の重複や空行を除去する（オプション）
	cleanedHistory := make([]string, 0, len(history))
	seen := make(map[string]struct{})
	for i := len(history) - 1; i >= 0; i-- { // 新しいものから見ていく
		line := history[i]
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}
		if _, ok := seen[trimmedLine]; !ok {
			cleanedHistory = append(cleanedHistory, trimmedLine) // 重複がなければ追加
			seen[trimmedLine] = struct{}{}
		}
	}
	// 順序を元に戻す
	for i, j := 0, len(cleanedHistory)-1; i < j; i, j = i+1, j-1 {
		cleanedHistory[i], cleanedHistory[j] = cleanedHistory[j], cleanedHistory[i]
	}

	// 履歴の最大件数を制限する（オプション）
	// const maxHistorySize = 1000
	// if len(cleanedHistory) > maxHistorySize {
	// 	cleanedHistory = cleanedHistory[len(cleanedHistory)-maxHistorySize:]
	// }

	return cleanedHistory
}

// saveHistory は履歴をファイルに書き込む
func saveHistory(filePath string, history []string) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		log.Printf("警告: 履歴ファイルの書き込みに失敗しました (%s): %v", filePath, err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	// 渡された履歴をそのまま書き込む (重複除去は loadHistory と executor で行う想定)
	for _, line := range history {
		// 空行は書き込まない
		if strings.TrimSpace(line) == "" {
			continue
		}
		if _, err := fmt.Fprintln(writer, line); err != nil {
			log.Printf("警告: 履歴の書き込み中にエラーが発生しました (%s): %v", filePath, err)
			return // エラーが発生したら中断
		}
	}

	if err := writer.Flush(); err != nil {
		log.Printf("警告: 履歴ファイルのフラッシュ中にエラーが発生しました (%s): %v", filePath, err)
	}
}
