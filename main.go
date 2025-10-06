// main.go
package main

import (
	"fmt"
	"getgoldprice/internal/getgoldprice"
	"log"
)

func main() {
	// 1) Headline (homepage)

	getprice := getgoldprice.GetGoldPrice{}
	head, err := getprice.GetLastPrice()
	if err != nil {
		log.Printf("headline error: %v", err)
	} else {
		//fmt.Printf("GTA Headline %s %s %s | Bar: Buy %.2f / Sell %.2f | Orn Sell %.2f | TaxBase %.2f\n",			head.DateBE, head.TimeTH, head.Seq, head.BarBuy, head.BarSell, head.JewelryBuy, head.JewelrySell)

		fmt.Printf("Release %s | Bar: Buy %.2f / Sell %.2f | Orn Sell %.2f | TaxBase %.2f\n",
			head.ReleaseAt, head.BarBuy, head.BarSell, head.JewelryBuy, head.JewelrySell)
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
