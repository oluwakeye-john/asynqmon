package asynqmon

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestUIAssetsHandlerServesViteBuildAtConfiguredRootPath(t *testing.T) {
	h := &uiAssetsHandler{
		rootPath:       "/monitoring",
		contents:       staticContents,
		staticDirPath:  "ui/build",
		indexFileName:  "index.html",
		prometheusAddr: "http://prometheus:9090",
		readOnly:       true,
	}

	indexRecorder := httptest.NewRecorder()
	h.ServeHTTP(
		indexRecorder,
		httptest.NewRequest(http.MethodGet, "http://example.com/monitoring/", nil),
	)

	if indexRecorder.Code != http.StatusOK {
		t.Fatalf("index status = %d, want %d", indexRecorder.Code, http.StatusOK)
	}
	indexBody := indexRecorder.Body.String()
	for _, want := range []string{
		`<base href="/monitoring/" />`,
		`window.FLAG_ROOT_PATH = "\/monitoring";`,
		`window.FLAG_READ_ONLY = "true";`,
	} {
		if !strings.Contains(indexBody, want) {
			t.Errorf("rendered index does not contain %q", want)
		}
	}

	assetPattern := regexp.MustCompile(`src="\./(assets/[^"]+\.js)"`)
	match := assetPattern.FindStringSubmatch(indexBody)
	if len(match) != 2 {
		t.Fatalf("could not find built JavaScript asset in rendered index")
	}

	assetRecorder := httptest.NewRecorder()
	h.ServeHTTP(
		assetRecorder,
		httptest.NewRequest(
			http.MethodGet,
			"http://example.com/monitoring/"+match[1],
			nil,
		),
	)

	if assetRecorder.Code != http.StatusOK {
		t.Fatalf("asset status = %d, want %d", assetRecorder.Code, http.StatusOK)
	}
	if got := assetRecorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/javascript") {
		t.Errorf("asset content type = %q, want application/javascript", got)
	}
}
