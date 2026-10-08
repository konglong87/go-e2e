//go:build computeracceptance && !windows

package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cu "github.com/konglong87/go-e2e/internal/computeruse"
)

func TestAcceptanceDisabledWithoutExplicitFlag(t *testing.T) {
	closeFn, err := startComputerAcceptance(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	closeFn()
	for _, args := range [][]string{{acceptanceSocketFlag}, {acceptanceSocketFlag, "relative"}, {acceptanceSocketFlag, "/a", acceptanceSocketFlag, "/b"}} {
		if _, err := acceptanceSocketArg(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
func acceptanceCall(h http.Handler, body string, headers bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "http://localhost/command", strings.NewReader(body))
	if headers {
		req.Header.Set(acceptanceHeader, "1")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
func TestAcceptanceRequestBoundary(t *testing.T) {
	m, _ := testComputerManager()
	h := acceptanceHandler(context.Background(), m)
	if r := acceptanceCall(h, `{"op":"start","approved":true}`, false); r.Code != http.StatusForbidden {
		t.Fatal(r.Code)
	}
	for _, body := range []string{`{"op":"start","extra":true}`, `{} {}`, strings.Repeat("x", acceptanceMaxBody+1)} {
		if r := acceptanceCall(h, body, true); r.Code != http.StatusBadRequest {
			t.Fatal(r.Code, r.Body.String())
		}
	}
	for _, origin := range []string{"null", "http://localhost", "https://evil.example"} {
		req := httptest.NewRequest(http.MethodPost, "http://localhost/command", strings.NewReader(`{"op":"start","approved":true}`))
		req.Header.Set(acceptanceHeader, "1")
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatal(rec.Code)
		}
	}
	if r := acceptanceCall(h, `{"op":"start"}`, true); !strings.Contains(r.Body.String(), "explicit acceptance session approval required") {
		t.Fatal(r.Body.String())
	}
	if m.controller != nil {
		t.Fatal("unapproved controller created")
	}
}
func TestAcceptanceUsesSharedManagerAndRejectsForgedBinding(t *testing.T) {
	m, _ := testComputerManager()
	ctx := context.Background()
	out, err := dispatchAcceptance(ctx, ctx, m, acceptanceRequest{Op: "start", Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	s := out.(ComputerSessionDTO)
	if s.ID != m.controller.Session().ID() {
		t.Fatal("separate authority")
	}
	for _, req := range []acceptanceRequest{
		{Op: "execute", SessionID: "other", Action: &cu.Action{SessionID: "other", Kind: cu.ActionClick}},
		{Op: "execute", SessionID: s.ID, Action: &cu.Action{SessionID: "other", Kind: cu.ActionClick}},
		{Op: "execute", SessionID: s.ID, Action: &cu.Action{SessionID: s.ID, Kind: cu.ActionStop}},
		{Op: "execute", SessionID: s.ID, Action: &cu.Action{ID: "no-observe", SessionID: s.ID, Kind: cu.ActionClick, Point: &cu.Point{X: 1, Y: 1}}},
		{Op: "execute", SessionID: s.ID}, {Op: "shell", SessionID: s.ID},
	} {
		if _, err := dispatchAcceptance(ctx, ctx, m, req); err == nil {
			t.Fatalf("accepted %+v", req)
		}
	}
	if _, err := dispatchAcceptance(ctx, ctx, m, acceptanceRequest{Op: "stop", SessionID: s.ID}); err != nil {
		t.Fatal(err)
	}
	if m.controller.Session().State() != cu.SessionStopped {
		t.Fatal("stop did not reach shared controller")
	}
}
func TestAcceptanceSocketPrivateAndLifecycle(t *testing.T) {
	// Darwin's temporary directory path may be a symlink and too long for a Unix
	// socket. Use a short private dir at /private/tmp for this non-secret test.
	dir, err := os.MkdirTemp("/tmp", "cu-sock-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "control.sock")
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := validateAcceptanceDirectory(path); err == nil {
		t.Fatal("public dir accepted")
	}
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	m, _ := testComputerManager()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	closeFn, err := startComputerAcceptance(ctx, m, []string{acceptanceSocketFlag, path})
	if err != nil {
		t.Fatal(err)
	}
	defer closeFn()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("mode %v %v", info, err)
	}
	if _, err := startComputerAcceptance(ctx, m, []string{acceptanceSocketFlag, path}); err == nil {
		t.Fatal("replaced existing socket")
	}
	tr := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", path)
	}}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: time.Second}
	req, _ := http.NewRequest(http.MethodPost, "http://localhost/command", strings.NewReader(`{"op":"start","approved":true}`))
	req.Header.Set(acceptanceHeader, "1")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var result struct {
		Data  ComputerSessionDTO `json:"data"`
		Error string             `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || result.Error != "" || result.Data.ID == "" {
		t.Fatalf("%s %v", raw, err)
	}
	closeFn()
	closeFn()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("socket not removed %v", err)
	}
}

func TestAcceptanceCancelledStartCannotApprove(t *testing.T) {
	m, count := testComputerManager()
	host := context.Background()
	ctx, cancel := context.WithCancel(host)
	m.mu.Lock()
	result := make(chan error, 1)
	go func() {
		_, err := dispatchAcceptance(host, ctx, m, acceptanceRequest{Op: "start", Approved: true})
		result <- err
	}()
	cancel()
	m.mu.Unlock()
	if err := <-result; err != context.Canceled {
		t.Fatalf("cancelled approval: %v", err)
	}
	if m.controller != nil || *count != 0 {
		t.Fatal("cancelled start created authority")
	}
}

func TestAcceptanceShutdownDrainsAndRejectsNewCommands(t *testing.T) {
	g := &acceptanceAdmission{}
	entered, release, drained := make(chan struct{}), make(chan struct{}), make(chan struct{})
	handler := g.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-release; w.WriteHeader(http.StatusOK) }))
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "http://localhost/command", nil))
	}()
	<-entered
	g.close()
	go func() { g.active.Wait(); close(drained) }()
	select {
	case <-drained:
		t.Fatal("shutdown did not wait for admitted handler")
	default:
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "http://localhost/command", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("post-close admission %d", rec.Code)
	}
	close(release)
	select {
	case <-drained:
	case <-time.After(time.Second):
		t.Fatal("handler drain stuck")
	}
	m, count := testComputerManager()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.startWithLifetime(context.Background(), ctx, ComputerSessionStartInput{Approved: true}); err != context.Canceled {
		t.Fatal(err)
	}
	if *count != 0 || m.controller != nil {
		t.Fatal("cancelled host resurrected backend")
	}
}

func TestAcceptanceTargetLaunchRequiresCurrentSessionAndRegisteredTarget(t *testing.T) {
	m, _ := testComputerManager()
	ctx := context.Background()
	out, err := dispatchAcceptance(ctx, ctx, m, acceptanceRequest{Op: "start", Approved: true})
	if err != nil {
		t.Fatal(err)
	}
	s := out.(ComputerSessionDTO)
	if _, err = dispatchAcceptance(ctx, ctx, m, acceptanceRequest{Op: "launch_target", SessionID: "forged", TargetID: "calculator"}); err == nil {
		t.Fatal("forged launch session accepted")
	}
	out, err = dispatchAcceptance(ctx, ctx, m, acceptanceRequest{Op: "launch_target", SessionID: s.ID, TargetID: "unregistered"})
	receipt, ok := out.(cu.LaunchReceipt)
	if err == nil || !ok || receipt.Outcome != cu.OutcomeRejected || receipt.ErrorCode != cu.ErrorCodeUnsupportedTarget {
		t.Fatalf("out=%+v err=%v", out, err)
	}
}
