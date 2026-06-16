package sei

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	// O número aproximado de órgãos no SEI-MG.
	orgaosLen = 80
)

// Scraper é responsável por extrair dados direto do página do SEI.
type Scraper struct {
	baseURL string
}

func NewScraper(baseURL string) *Scraper {
	return &Scraper{
		baseURL: baseURL,
	}
}

type Orgao struct {
	ID   int
	Nome string
}

// ListOrgaos lista os órgãos disponíveis no SEI na página de login.
func (s *Scraper) ListOrgaos(ctx context.Context) ([]Orgao, error) {
	res, err := http.Get(s.baseURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	orgaos := make([]Orgao, 0, orgaosLen)
	doc.Find("#selOrgao > option").Each(func(i int, s *goquery.Selection) {
		value, ok := s.Attr("value")
		if !ok {
			return
		}

		// Pegamos apenas valores inteiros válidos.
		id, err := strconv.Atoi(value)
		if err != nil {
			return
		}

		orgaos = append(orgaos, Orgao{
			ID:   id,
			Nome: strings.TrimSpace(s.Text()),
		})
	})

	return orgaos, nil
}
