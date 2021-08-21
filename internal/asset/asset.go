package asset

import (
	q "github.com/achannarasappa/ticker/internal/adapter/yahoo"
	c "github.com/achannarasappa/ticker/internal/common"
	"github.com/achannarasappa/ticker/internal/currency"
)

type AggregatedLot struct {
	Symbol     string
	Cost       float64
	Quantity   float64
	OrderIndex int
}

func GetAssets(dep c.Dependencies, ctx c.Context) []c.Asset {

	assets := getAssetsFixed(ctx)
	symbols := getSymbols(ctx.Config)
	assetQuotes := q.GetAssetQuotes(*dep.HttpClient, symbols)
	lotsBySymbol := getLots(ctx.Config.Lots)

	for i, assetQuote := range assetQuotes {

		currencyRateByUse := currency.GetCurrencyRateFromContext(ctx, assetQuote.Currency.Code)
		currencyCode := currencyRateByUse.ToCurrencyCode

		holding := getHolding(assetQuote, lotsBySymbol)

		assets = append(assets, c.Asset{
			Name:   assetQuote.Name,
			Symbol: assetQuote.Symbol,
			Class:  assetQuote.Class,
			Currency: c.Currency{
				Code:          assetQuote.Currency.Code,
				CodeConverted: currencyCode,
			},
			Holding:       convertAssetHoldingCurrency(currencyRateByUse, holding),
			QuotePrice:    convertAssetQuotePriceCurrency(currencyRateByUse, assetQuote.QuotePrice),
			QuoteExtended: convertAssetQuoteExtendedCurrency(currencyRateByUse, assetQuote.QuoteExtended),
			Exchange:      assetQuote.Exchange,
			Meta: c.Meta{
				IsVariablePrecision: false,
				OrderIndex:          i,
			},
		})

	}

	// combine asset quotes, lots, and holdings to produce assets

	return []c.Asset{}

}

func getHolding(assetQuote c.AssetQuote, lotsBySymbol map[string]AggregatedLot) c.Holding {

	if aggregatedLot, ok := lotsBySymbol[assetQuote.Symbol]; ok {
		value := aggregatedLot.Quantity * assetQuote.QuotePrice.Price
		totalChangeAmount := value - aggregatedLot.Cost
		totalChangePercent := (totalChangeAmount / aggregatedLot.Cost) * 100

		return c.Holding{
			Value:     value,
			Cost:      aggregatedLot.Cost,
			Quantity:  aggregatedLot.Quantity,
			UnitValue: value / aggregatedLot.Quantity,
			UnitCost:  aggregatedLot.Cost / aggregatedLot.Quantity,
			DayChange: c.HoldingChange{
				Amount:  assetQuote.QuotePrice.Change * aggregatedLot.Quantity,
				Percent: assetQuote.QuotePrice.ChangePercent,
			},
			TotalChange: c.HoldingChange{
				Amount:  totalChangeAmount,
				Percent: totalChangePercent,
			},
			Weight: 0,
		}
	}

	return c.Holding{}

}

func getSymbols(config c.Config) []string {

	symbols := make(map[string]bool)
	symbolsUnique := make([]string, 0)

	for _, symbol := range config.Watchlist {
		if !symbols[symbol] {
			symbols[symbol] = true
			symbolsUnique = append(symbolsUnique, symbol)
		}
	}

	if config.ShowHoldings {
		for _, lot := range config.Lots {
			if !symbols[lot.Symbol] {
				symbols[lot.Symbol] = true
				symbolsUnique = append(symbolsUnique, lot.Symbol)
			}
		}
	}

	return symbolsUnique

}

func getLots(lots []c.Lot) map[string]AggregatedLot {

	if lots == nil {
		return map[string]AggregatedLot{}
	}

	aggregatedLots := map[string]AggregatedLot{}

	for i, lot := range lots {

		aggregatedLot, ok := aggregatedLots[lot.Symbol]

		if !ok {

			aggregatedLots[lot.Symbol] = AggregatedLot{
				Symbol:     lot.Symbol,
				Cost:       (lot.UnitCost * lot.Quantity) + lot.FixedCost,
				Quantity:   lot.Quantity,
				OrderIndex: i,
			}

		} else {

			aggregatedLot.Quantity = aggregatedLot.Quantity + lot.Quantity
			aggregatedLot.Cost = aggregatedLot.Cost + (lot.Quantity * lot.UnitCost)

			aggregatedLots[lot.Symbol] = aggregatedLot

		}

	}

	return aggregatedLots
}