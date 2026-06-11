package conteudo

import "github.com/tmc/langchaingo/textsplitter"

const (
	// chunkSize é o tamanho máximo de um chunk em caracteres.
	chunkSize = 1500
	// chunkOverlap é a sobreposição entre chunks consecutivos, preservando
	// contexto nas bordas.
	chunkOverlap = 200
)

// splitText divide o texto em chunks com sobreposição, escolhendo a estratégia
// de acordo com o formato do conteúdo. Conteúdo em Markdown é dividido
// preservando a estrutura do documento; os demais formatos são divididos
// recursivamente por caracteres.
func splitText(text, formato string) ([]string, error) {
	var splitter textsplitter.TextSplitter
	switch formato {
	case FormatoMarkdown:
		splitter = textsplitter.NewMarkdownTextSplitter(
			textsplitter.WithChunkSize(chunkSize),
			textsplitter.WithChunkOverlap(chunkOverlap),
		)
	default:
		splitter = textsplitter.NewRecursiveCharacter(
			textsplitter.WithChunkSize(chunkSize),
			textsplitter.WithChunkOverlap(chunkOverlap),
		)
	}
	return splitter.SplitText(text)
}
