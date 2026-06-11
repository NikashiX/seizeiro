package markdown

import (
	"strings"
	"testing"
)

func TestConvertHTML(t *testing.T) {
	t.Parallel()

	html := `<html><body><h1>Titulo</h1><p>Paragrafo com <strong>negrito</strong>.</p></body></html>`

	got, err := ConvertHTML(strings.NewReader(html), "text/html; charset=utf-8")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, "# Titulo") {
		t.Fatalf("expected markdown heading, got:\n%s", got)
	}
	if !strings.Contains(got, "**negrito**") {
		t.Fatalf("expected bold markdown, got:\n%s", got)
	}
}

func TestConvertHTML_WithoutImg(t *testing.T) {
	t.Parallel()

	html := `<html><body><p>Texto</p><img src="foto.png" alt="foto"><p>Fim</p></body></html>`

	withImages, err := ConvertHTML(strings.NewReader(html), "text/html; charset=utf-8")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(withImages, "foto") {
		t.Fatalf("expected image reference in default output, got:\n%s", withImages)
	}

	WithoutImg, err := ConvertHTML(strings.NewReader(html), "text/html; charset=utf-8", WithoutImg())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(WithoutImg, "foto") {
		t.Fatalf("expected no image reference with WithoutImg, got:\n%s", WithoutImg)
	}
	if !strings.Contains(WithoutImg, "Texto") || !strings.Contains(WithoutImg, "Fim") {
		t.Fatalf("expected text content preserved, got:\n%s", WithoutImg)
	}
}
