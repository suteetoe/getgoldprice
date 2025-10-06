// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const PRICE_AT_DATE_TAGID = "DetailPlace_uc_goldprices1_lblAsTime"
const PRICE_BL_SELL_TAGID = "DetailPlace_uc_goldprices1_lblBLSell"
const PRICE_BL_BUY_TAGID = "DetailPlace_uc_goldprices1_lblBLBuy"
const PRICE_OM_SELL_TAGID = "DetailPlace_uc_goldprices1_lblOMSell"
const PRICE_OM_BUY_TAGID = "DetailPlace_uc_goldprices1_lblOMBuy"

type Headline struct {
	DateBE      string  // e.g. "04/10/2568"
	TimeTH      string  // e.g. "09:07 น."
	Seq         string  // e.g. "(ครั้งที่ 1)"
	BarSell     float64 // ทองคำแท่ง 96.5% ขายออก
	BarBuy      float64 // ทองคำแท่ง 96.5% รับซื้อ
	JewelrySell float64 // ทองรูปพรรณ 96.5% ขายออก
	JewelryBuy  float64 // ฐานภาษี
	Src         string  // source URL
	Fetched     time.Time
}

func client() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

func fetchDoc(url string) (*goquery.Document, error) {
	req, _ := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	req.Header.Set("User-Agent", "GTA-GoldScraper/1.1 (+https://example.com; contact: dev@example.com)")
	resp, err := client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return goquery.NewDocumentFromReader(resp.Body)
}

func asFloat(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseBEtoCE(dateBE string) string {
	// quick helper if you want CE display (keep BE in the struct if you prefer)
	parts := strings.Split(strings.TrimSpace(dateBE), "/")
	if len(parts) != 3 {
		return ""
	}
	// yearBE, _ := strconv.Atoi(parts[2])
	// yearCE := yearBE - 543
	return fmt.Sprintf("%s-%s-%02d", parts[2], parts[1], 0) + fmt.Sprintf("") // not used; kept minimal
}

// ========== 1) HOMEPAGE DOM ==========
// Scrapes the “Gold Price by GTA / ราคาทองตามประกาศของสมาคมค้าทองคำ” block by DOM.
func scrapeHomepage() (*Headline, error) {
	const url = "https://www.goldtraders.or.th/"
	doc, err := fetchDoc(url)
	if err != nil {
		return nil, err
	}

	priceAtTxt := strings.TrimSpace(doc.Find(fmt.Sprintf("span#%s", PRICE_AT_DATE_TAGID)).Text())
	sellTxt := strings.TrimSpace(doc.Find(fmt.Sprintf(`span#%s`, PRICE_BL_SELL_TAGID)).Text())
	buyTxt := strings.TrimSpace(doc.Find(fmt.Sprintf(`span#%s`, PRICE_BL_BUY_TAGID)).Text())
	jwSellTxr := strings.TrimSpace(doc.Find(fmt.Sprintf(`span#%s`, PRICE_OM_SELL_TAGID)).Text())
	jwBuyTxt := strings.TrimSpace(doc.Find(fmt.Sprintf(`span#%s`, PRICE_OM_BUY_TAGID)).Text())

	fmt.Printf("at: %s buy: %s sell: %s, jwSell: %s, jwBuy: %s\n", priceAtTxt, sellTxt, buyTxt, jwSellTxr, jwBuyTxt)
	var h Headline
	h.Src = url
	h.Fetched = time.Now()

	/*
		// Find the section heading, then walk its container to pick values.
		// The page displays lines like:
		//   ประจำวันที่ 04/10/2568 เวลา 09:07 น. (ครั้งที่ 1)
		//   ทองคำแท่ง 96.5%  ขายออก  59,450.00
		//    รับซื้อ  59,350.00
		//   ทองรูปพรรณ 96.5%  ขายออก  60,250.00
		//    ฐานภาษี  58,168.92
		// We’ll search for the heading then read its nearby text nodes.
		section := doc.Find("h3, h2").FilterFunction(func(i int, s *goquery.Selection) bool {
			txt := strings.TrimSpace(s.Text())
			return strings.Contains(txt, "Gold Price by GTA") || strings.Contains(txt, "ราคาทองตามประกาศของสมาคมค้าทองคำ")
		}).First()
		if section.Length() == 0 {
			return nil, fmt.Errorf("headline section not found")
		}

		// The block is typically plain text siblings (no stable IDs), so collect the next few nodes.
		container := section.Parent()
		lines := strings.FieldsFunc(container.Text(), func(r rune) bool { return r == '\n' || r == '\r' })
		// Compact & keep only non-empty
		mk := make([]string, 0, len(lines))
		for _, t := range lines {
			t = strings.TrimSpace(strings.ReplaceAll(t, "\u00a0", " "))
			if t != "" {
				mk = append(mk, t)
			}
		}

		// Pull key fields by scanning tokens
		var h Headline
		h.Src = url
		h.Fetched = time.Now()

		// date/time/seq line starts with "ประจำวันที่"
		for i := 0; i < len(mk); i++ {
			if strings.HasPrefix(mk[i], "ประจำวันที่") {
				// example: "ประจำวันที่ 04/10/2568 เวลา 09:07 น. (ครั้งที่ 1)"
				// pick the date token right after "ประจำวันที่"
				if i+1 < len(mk) {
					h.DateBE = mk[i+1]
				}
				// search forward for "เวลา" and seq "(ครั้งที่"
				for j := i + 1; j < len(mk) && j < i+12; j++ {
					if mk[j] == "เวลา" && j+1 < len(mk) {
						h.TimeTH = mk[j+1]
					}
					if strings.HasPrefix(mk[j], "(ครั้งที่") {
						h.Seq = mk[j]
						break
					}
				}
				break
			}
		}

		// scan for price labels and numbers that follow them
		for i := 0; i < len(mk); i++ {
			switch mk[i] {
			case "ขายออก":
				// last “ทองคำแท่ง 96.5% ขายออก <num>”
				if i+1 < len(mk) && h.BarSell == 0 {
					h.BarSell = asFloat(mk[i+1])
				} else if i+1 < len(mk) && h.OrnSell == 0 && h.BarSell > 0 {
					// next "ขายออก" typically belongs to ornament (ทองรูปพรรณ)
					h.OrnSell = asFloat(mk[i+1])
				}
			case "รับซื้อ":
				if i+1 < len(mk) {
					h.BarBuy = asFloat(mk[i+1])
				}
			case "ฐานภาษี":
				if i+1 < len(mk) {
					h.TaxBase = asFloat(mk[i+1])
				}
			}
		}
	*/

	return &h, nil
}

// // ========== 2) INTRADAY DOM ==========
// // Scrapes the archive/intraday table by walking <table> rows.
// // Falls back to reading the text container if <table> structure changes.
// func scrapeIntradayDOM() ([]IntradayRow, error) {
// 	const url = "https://www.goldtraders.or.th/TaxarchiveList.aspx"
// 	doc, err := fetchDoc(url)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var out []IntradayRow

// 	// Try to find a real table with a header containing "วันที่" (Date)
// 	table := doc.Find("table").FilterFunction(func(i int, s *goquery.Selection) bool {
// 		// look for a header cell that contains "วันที่"
// 		haveDate := s.Find("th,td").FilterFunction(func(i int, h *goquery.Selection) bool {
// 			return strings.Contains(h.Text(), "วันที่")
// 		}).Length() > 0
// 		return haveDate
// 	}).First()

// 	if table.Length() > 0 {
// 		// Walk rows, skip header
// 		table.Find("tr").Each(func(i int, tr *goquery.Selection) {
// 			cells := tr.Find("th,td")
// 			if cells.Length() < 4 {
// 				return
// 			}
// 			// Expected order: วันที่, (ครั้งที่), ทองคำแท่งรับซื้อ, ทองคำแท่งขายออก, ...
// 			dateBE := strings.TrimSpace(cells.Eq(0).Text())
// 			seqTxt := strings.TrimSpace(cells.Eq(1).Text())
// 			buyTxt := strings.TrimSpace(cells.Eq(2).Text())
// 			sellTxt := strings.TrimSpace(cells.Eq(3).Text())

// 			if dateBE == "" || buyTxt == "" || sellTxt == "" {
// 				return
// 			}
// 			seqTxt = strings.Trim(seqTxt, "()ครั้งที่ ")
// 			seq, _ := strconv.Atoi(seqTxt)

// 			out = append(out, IntradayRow{
// 				DateBE: dateBE,
// 				Seq:    seq,
// 				Buy:    asFloat(buyTxt),
// 				Sell:   asFloat(sellTxt),
// 				Src:    url,
// 			})
// 		})
// 		return out, nil
// 	}

// 	// Fallback: the page sometimes renders the rows as text blocks without explicit <td>s.
// 	// In that case we locate the year/month section and split by lines.
// 	yearBlock := doc.Find("body").First().Text()
// 	lines := strings.Split(yearBlock, "\n")
// 	for _, ln := range lines {
// 		ln = strings.TrimSpace(strings.ReplaceAll(ln, "\u00a0", " "))
// 		// lines look like: "04/10/2568 (1) 59,350.00 59,450.00 ..."
// 		if len(ln) < 20 || !strings.Contains(ln, "/") || !strings.Contains(ln, "(") {
// 			continue
// 		}
// 		parts := strings.Fields(ln)
// 		// parts[0] = dateBE, parts[1] = "(1)", parts[2] = buy, parts[3] = sell
// 		if len(parts) >= 4 && strings.Count(parts[0], "/") == 2 && strings.HasPrefix(parts[1], "(") {
// 			seq := strings.Trim(parts[1], "()")
// 			seqN, _ := strconv.Atoi(seq)
// 			out = append(out, IntradayRow{
// 				DateBE: parts[0],
// 				Seq:    seqN,
// 				Buy:    asFloat(parts[2]),
// 				Sell:   asFloat(parts[3]),
// 				Src:    url,
// 			})
// 		}
// 	}
// 	return out, nil
// }

func main() {
	// 1) Headline (homepage)
	head, err := scrapeHomepage()
	if err != nil {
		log.Printf("headline error: %v", err)
	} else {
		fmt.Printf("GTA Headline %s %s %s | Bar: Buy %.2f / Sell %.2f | Orn Sell %.2f | TaxBase %.2f\n",
			head.DateBE, head.TimeTH, head.Seq, head.BarBuy, head.BarSell, head.JewelryBuy, head.JewelrySell)
	}

	// // 2) Intraday (archive list)
	// rows, err := scrapeIntradayDOM()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("Intraday rows parsed: %d\n", len(rows))
	// for i := 0; i < len(rows) && i < 5; i++ {
	// 	r := rows[i]
	// 	fmt.Printf("%s (Seq %d): Buy %.2f | Sell %.2f\n", r.DateBE, r.Seq, r.Buy, r.Sell)
	// }
}
