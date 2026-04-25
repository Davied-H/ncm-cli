package desktop

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
)

type orpheusPayload struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
	Cmd  string `json:"cmd"`
}

func SongPlayURL(id int64) (string, error) {
	if id <= 0 {
		return "", fmt.Errorf("song-id 必须大于 0")
	}
	payload, err := json.Marshal(orpheusPayload{
		Type: "song",
		ID:   id,
		Cmd:  "play",
	})
	if err != nil {
		return "", err
	}
	return "orpheus://" + base64.StdEncoding.EncodeToString(payload), nil
}

func Open(url string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("当前只支持 macOS 网易云音乐桌面端 URL Scheme")
	}
	if url == "" {
		return fmt.Errorf("URL 不能为空")
	}
	return exec.Command("open", url).Run()
}
