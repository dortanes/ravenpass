package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/app/preferences"
)

// recordingPrinter keeps each page it is given and whether the host was held while it printed.
type recordingPrinter struct {
	held  func() bool
	jobs  []string
	pages []string
	holds []bool
	err   error
}

func (p *recordingPrinter) Print(job string, page []byte) error {
	p.jobs = append(p.jobs, job)
	p.pages = append(p.pages, string(page))
	p.holds = append(p.holds, p.held())
	return p.err
}

func newPrintingService(t *testing.T) (*Service, *recordingPrinter) {
	t.Helper()
	home := t.TempDir()
	service, files := newServiceWithStorage(t, filepath.Join(home, "storage.json"), filepath.Join(home, "vault.rpv"))
	if err := files.Open(); err != nil {
		t.Fatal(err)
	}
	holding := false
	service.hold = func() func() {
		holding = true
		return func() { holding = false }
	}
	printer := &recordingPrinter{held: func() bool { return holding }}
	service.printer = printer
	return service, printer
}

func TestTheStagedRecoveryKeyPrintsItsWordsInOrder(t *testing.T) {
	service, printer := newPrintingService(t)
	phrase, err := service.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	if err := service.PrintRecoveryKey(phrase); err != nil {
		t.Fatal(err)
	}
	if len(printer.pages) != 1 {
		t.Fatalf("pages printed = %d, want 1", len(printer.pages))
	}
	sheet := service.preferences.RecoverySheet()
	if printer.jobs[0] != sheet.Title || !printer.holds[0] {
		t.Fatalf("job = %q, held = %t", printer.jobs[0], printer.holds[0])
	}
	page := printer.pages[0]
	words := strings.Fields(phrase)
	if len(words) != 24 {
		t.Fatalf("staged phrase has %d words", len(words))
	}
	from := 0
	for i, word := range words {
		cell := fmt.Sprintf("<li><span>%d</span>%s</li>", i+1, word)
		at := strings.Index(page[from:], cell)
		if at < 0 {
			t.Fatalf("word %d is missing or out of order", i+1)
		}
		from += at + len(cell)
	}
	for _, line := range []string{sheet.Keep, sheet.Warning, time.Now().Format(time.DateOnly)} {
		if !strings.Contains(page, line) {
			t.Fatalf("page is missing %q", line)
		}
	}
	if strings.Contains(page, "<script") || strings.Contains(page, "src=") || strings.Contains(page, "href=") {
		t.Fatal("the page loads a resource or runs a script")
	}
}

func TestNothingPrintsWithoutTheStagedRecoveryKey(t *testing.T) {
	service, printer := newPrintingService(t)
	assertFailure(t, service.PrintRecoveryKey("abandon ability able"), failureSetupNotActive)
	phrase, err := service.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	assertFailure(t, service.PrintRecoveryKey("other words"), failureRecoveryKeyMismatch)
	if err := service.Lock(); err != nil {
		t.Fatal(err)
	}
	assertFailure(t, service.PrintRecoveryKey(phrase), failureSetupNotActive)
	if len(printer.pages) != 0 {
		t.Fatal("a recovery key that is not staged reached the printer")
	}
}

func TestAFailedPrintIsReported(t *testing.T) {
	service, printer := newPrintingService(t)
	printer.err = errors.New("no printer")
	phrase, err := service.BeginCreation()
	if err != nil {
		t.Fatal(err)
	}
	assertFailure(t, service.PrintRecoveryKey(phrase), failureRecoveryKeyPrintFailed)
}

func TestOnlyAHostWithAPrinterOffersPrinting(t *testing.T) {
	if newServiceOnHost(t, Host{}).Capabilities().Print {
		t.Fatal("a host without a printer offers printing")
	}
	if !newServiceOnHost(t, Host{Printer: &recordingPrinter{}}).Capabilities().Print {
		t.Fatal("a host with a printer does not offer printing")
	}
}

func TestTheRecoverySheetEscapesItsText(t *testing.T) {
	text := preferences.RecoverySheetText{Language: "en", Title: "<b>", Printed: "on {date}", Keep: "&", Warning: "\""}
	page, err := renderRecoverySheet("one <two>", text, time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<title>&lt;b&gt;</title>", "on 2026-09-27", "<span>2</span>&lt;two&gt;", "<p>&amp;</p>", "<p>&#34;</p>"} {
		if !strings.Contains(string(page), want) {
			t.Fatalf("page is missing %q", want)
		}
	}
}

func TestRecoveryFileIsPrivateVerifiedAndReplaceable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recovery.txt")
	phrase := "private words that must stay on this device"
	english := newTestPreferences(t).RecoveryFile()
	if err := writeRecoveryKey(savedPath{path: path, sandbox: unconfined{}}, phrase, english); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		t.Fatalf("recovery file mode = %v", info.Mode())
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), phrase) || !strings.Contains(string(first), "does not contain your passwords") || !strings.Contains(string(first), "This file is not encrypted. Keep it private.") {
		t.Fatal("recovery file is missing its key or handling notice")
	}
	if err := writeRecoveryKey(savedPath{path: path, sandbox: unconfined{}}, "replacement words", english); err != nil {
		t.Fatalf("replace error = %v", err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(second), "replacement words") {
		t.Fatal("confirmed replacement did not reach the chosen file")
	}
	info, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		t.Fatalf("replaced recovery file mode = %v", info.Mode())
	}
}

func TestRecoveryClipboardClearsOnlyItsCurrentCopy(t *testing.T) {
	service := &Service{preferences: newTestPreferences(t)}
	clipboard := attachPasteboard(service)
	var clearFirst, clearSecond func()
	scheduleFirst := func(after time.Duration, clear func()) {
		if after != preferences.DefaultClipboardClearing().After {
			t.Fatalf("clear delay = %v", after)
		}
		clearFirst = clear
	}
	if err := service.copyRecoveryKey("first recovery phrase", scheduleFirst); err != nil {
		t.Fatal(err)
	}
	if clipboard.text != "first recovery phrase" {
		t.Fatal("phrase was not copied")
	}
	if err := service.copyRecoveryKey("second recovery phrase", func(_ time.Duration, clear func()) { clearSecond = clear }); err != nil {
		t.Fatal(err)
	}
	clearFirst()
	if clipboard.text != "second recovery phrase" {
		t.Fatal("older timer cleared a newer copy")
	}
	clipboard.replace("something else")
	clearSecond()
	if clipboard.text != "something else" {
		t.Fatal("changed clipboard content was cleared")
	}
	if err := service.copyRecoveryKey("third recovery phrase", func(_ time.Duration, clear func()) { clearSecond = clear }); err != nil {
		t.Fatal(err)
	}
	clearSecond()
	if clipboard.text != "" {
		t.Fatal("unchanged recovery phrase remained in clipboard")
	}
}

func TestRecoveryClipboardWriteFailureIsSafe(t *testing.T) {
	service := &Service{preferences: newTestPreferences(t)}
	clipboard := attachPasteboard(service)
	clipboard.failing = true
	scheduled := false
	if err := service.copyRecoveryKey("secret phrase", func(time.Duration, func()) { scheduled = true }); err == nil {
		t.Fatal("clipboard write failure was ignored")
	}
	if scheduled || clipboard.text != "" {
		t.Fatal("failed copy scheduled clear or changed clipboard")
	}
}
