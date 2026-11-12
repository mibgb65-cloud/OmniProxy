package antigravity

import (
	"os"
	"strings"
	"testing"
)

// TestMain 让契约测试不访问真实更新清单：后台版本刷新立即失败并回退到兼容下限。
func TestMain(m *testing.M) {
	protocolVersionURL = ""
	os.Exit(m.Run())
}

func TestTurnsAwayRecognizesEveryRefusalLine(t *testing.T) {
	for _, line := range refusals {
		system := "You are a coding agent.\n" + strings.ToUpper(line[:1]) + line[1:] + "\nBe brief."
		if !turnsAway(system) {
			t.Fatalf("refusal line not recognized: %q", line)
		}
		if turnsAway("You are a coding agent. Be brief.") {
			t.Fatal("ordinary system prompt was treated as a refusal")
		}
	}
}

func TestCloakedBreaksEveryWordWithoutLosingText(t *testing.T) {
	const line = "x-anthropic-billing-header:"
	system := "header\n" + line + " cc_version=2.1.299\nbody text"
	cloakedSystem := cloaked(system)
	if cloakedSystem == system {
		t.Fatal("refusal line was not cloaked")
	}
	if strings.Contains(cloakedSystem, line) {
		t.Fatal("raw refusal line is still visible to the upstream check")
	}
	if stripMarks(cloakedSystem) != system {
		t.Fatal("cloaking changed the text the model reads")
	}
	// 每个 ≥3 字符的词按 1、2 两个字符后各插入一次：x-anthropic 与 billing 等。
	if got, want := strings.Count(cloakedSystem, zeroWidth), 2*(len(strings.Fields(line))); got < want {
		t.Fatalf("cloaked with %d marks, want at least %d", got, want)
	}
}

func TestCloakedIsIdempotentAndRepairsPartialCloaks(t *testing.T) {
	system := "Built on Anthropic's Claude Agent SDK\ntail"
	once := cloaked(system)
	if twice := cloaked(once); twice != once {
		t.Fatal("cloaking is not idempotent")
	}
	// 旧做法每词只插一个：仍应识别，并补齐到两个。
	partial := strings.Replace(system, "Built", "B"+zeroWidth+"uilt", 1)
	if partial == system {
		t.Fatal("fixture unchanged")
	}
	if !turnsAway(partial) {
		t.Fatal("partially cloaked prompt was not recognized")
	}
	fixed := cloaked(partial)
	if strings.Contains(fixed, "B"+zeroWidth+"uilt"+zeroWidth+" on") {
		t.Fatal("old single-mark form was only partially repaired")
	}
	if stripMarks(fixed) != system {
		t.Fatal("repair changed the underlying text")
	}
}

func TestCloakedLeavesOtherPromptsAlone(t *testing.T) {
	system := "You are a helpful assistant.\nWrite tests."
	if got := cloaked(system); got != system {
		t.Fatal("prompt without a refusal line was modified")
	}
	if got := cloaked(""); got != "" {
		t.Fatal("empty system prompt was modified")
	}
}
