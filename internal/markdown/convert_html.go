package markdown

import (
	"io"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/strikethrough"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"golang.org/x/net/html/charset"
)

type OptionFunc func(*converter.Converter)

// WithoutImg remove todas as tags <img> do HTML antes de converter para Markdown.
func WithoutImg() OptionFunc {
	return func(conv *converter.Converter) {
		conv.Register.TagType("img", converter.TagTypeRemove, converter.PriorityStandard)
	}
}

// ConvertHTML converte um [io.Reader] de HTML para Markdown.
// O Content-Type deve ser informado uma vez que o conversor utiliza
// [charset.NewReader] para detectar a codificação.
func ConvertHTML(r io.Reader, contentType string, opts ...OptionFunc) (string, error) {
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			strikethrough.NewStrikethroughPlugin(),
			table.NewTablePlugin(),
		),
	)
	for _, fn := range opts {
		fn(conv)
	}

	rd, err := charset.NewReader(r, contentType)
	if err != nil {
		return "", err
	}

	md, err := conv.ConvertReader(rd)
	if err != nil {
		return "", err
	}
	return string(md), nil
}
