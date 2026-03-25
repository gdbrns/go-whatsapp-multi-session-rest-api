package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type moduleInfo struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
	Dir     string `json:"Dir"`
}

func main() {
	info, err := loadWhatsmeowModuleInfo()
	if err != nil {
		fatalf("resolve whatsmeow module: %v", err)
	}

	required := []string{
		"MessageType     *waAICommonDeprecated.AIRichResponseMessageType",
		"Submessages     []*waAICommonDeprecated.AIRichResponseSubMessage",
		"func (x *AIRichResponseMessage) GetMessageType() waAICommonDeprecated.AIRichResponseMessageType",
		"func (x *AIRichResponseMessage) GetSubmessages() []*waAICommonDeprecated.AIRichResponseSubMessage",
		"(waAICommonDeprecated.AIRichResponseMessageType)(0)",
		"(*waAICommonDeprecated.AIRichResponseSubMessage)(nil)",
		"264, // 341: WAWebProtobufsE2E.AIRichResponseMessage.messageType:type_name -> WAAICommonDeprecated.AIRichResponseMessageType",
		"265, // 342: WAWebProtobufsE2E.AIRichResponseMessage.submessages:type_name -> WAAICommonDeprecated.AIRichResponseSubMessage",
	}

	localPath := "proto/waE2E/WAWebProtobufsE2E.pb.go"
	localBytes, err := os.ReadFile(localPath)
	failed := false
	if err != nil {
		failed = true
		fmt.Fprintf(os.Stderr, "missing local file %s: %v\n", localPath, err)
	} else {
		localText := string(localBytes)
		for _, needle := range required {
			if !strings.Contains(localText, needle) {
				failed = true
				fmt.Fprintf(os.Stderr, "proto mismatch: %s is missing %q\n", localPath, needle)
			}
		}
	}

	if failed {
		os.Exit(1)
	}

	fmt.Printf("whatsmeow AIRichResponse bindings match %s\n", info.Version)
}

func loadWhatsmeowModuleInfo() (*moduleInfo, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "go.mau.fi/whatsmeow")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var info moduleInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, err
	}

	if info.Path == "" {
		return nil, errors.New("missing module path")
	}
	if info.Dir == "" {
		return nil, errors.New("missing module directory")
	}
	if info.Version == "" {
		info.Version = "unknown"
	}

	return &info, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
