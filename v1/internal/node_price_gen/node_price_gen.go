/**
 * Node Price Generator contains the logic to generate random prices
 * In reality this is replaced with an upstream API for doing this
 **/

package internal

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"math/rand"
	"sync"
	"time"

	helpers "github.com/hongkongkiwi/go-currency-nodes/v1/internal/helpers"
)

var seedRand *rand.Rand

var UpdatesPaused = false

type PriceGeneratorApi struct {
	currencyPairs []string
	latestPrices  map[string]*PriceCurrency
	updatesChan   chan<- map[string]*PriceCurrency
}

type PriceCurrency struct {
	Price       float64
	GeneratedAt time.Time
}

// func funcName() string {
// 	pc, _, _, _ := runtime.Caller(1)
// 	names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
// 	return names[len(names)-1]
// }

func genRandPrice(lower, upper float64) float64 {
	return truncate(lower+math.Abs(seedRand.Float64())*(upper-lower), lower)
}

func genRandWithinPercent(currPrice float64, percent uint) float64 {
	upperValue := math.Max(0.01, currPrice*(1+float64(percent)/100))
	lowerValue := math.Max(0.01, currPrice*(1-float64(percent)/100))
	return genRandPrice(lowerValue, upperValue)
}

// Correctly truncate a float64
// https://stackoverflow.com/questions/54362751/how-can-i-truncate-float64-number-to-a-particular-precision
func truncate(f float64, unit float64) float64 {
	bf := big.NewFloat(0).SetPrec(1000).SetFloat64(f)
	bu := big.NewFloat(0).SetPrec(1000).SetFloat64(unit)
	bf.Quo(bf, bu)
	// Truncate:
	i := big.NewInt(0)
	bf.Int(i)
	bf.SetInt(i)
	f, _ = bf.Mul(bf, bu).Float64()
	return f
}

func NewPriceCurrency(setPrice float64) (*PriceCurrency, error) {
	if setPrice <= 0.01 {
		setPrice = genRandPrice(0.01, 100)
	}
	pc := &PriceCurrency{Price: setPrice, GeneratedAt: time.Now()}
	return pc, nil
}

func NewPriceCurrencyRand() (*PriceCurrency, error) {
	return NewPriceCurrency(0)
}

// Constructor which init's the currency pairs
func NewPriceGeneratorApi(currencyPairs []string, updatesChan chan<- map[string]*PriceCurrency) (*PriceGeneratorApi, error) {
	pg := &PriceGeneratorApi{
		latestPrices:  make(map[string]*PriceCurrency),
		currencyPairs: currencyPairs,
		updatesChan:   updatesChan,
	}
	seedRand = rand.New(rand.NewSource(time.Now().Unix()))
	for _, currencyPair := range pg.currencyPairs {
		newPriceCurrency, _ := NewPriceCurrencyRand()
		pg.latestPrices[currencyPair] = newPriceCurrency
	}
	return pg, nil
}

func removeDuplicateInt(intSlice []int) []int {
	allKeys := make(map[int]bool)
	list := []int{}
	for _, item := range intSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

// Take a random selection of currency pairs and updates them by a percentage
func (pg *PriceGeneratorApi) GenRandPrices(percent uint) {
	// If we have no currency pairs then we cannot generate any prices
	if len(pg.currencyPairs) == 0 {
		return
	}
	// Generate a random set of indexes of our currency pairs
	var indexes []int
	for i := 0; i < seedRand.Intn(len(pg.currencyPairs))+1; i++ {
		// Not efficient but currency pairs are small so it does not matter
		indexes = append(indexes, seedRand.Intn(len(pg.currencyPairs)))
	}
	updatedPrices := make(map[string]*PriceCurrency)

	// Remove duplicates of indexes (we only need to update each currency pair once)
	indexes = removeDuplicateInt(indexes)
	for i := range indexes {
		currencyPair := pg.currencyPairs[i : i+1][0]
		existingPriceCurrency := pg.latestPrices[currencyPair]
		// Save a new PriceCurrency item into our latest prices array
		newPriceCurrency, _ := NewPriceCurrency(genRandWithinPercent(existingPriceCurrency.Price, percent))
		pg.latestPrices[currencyPair] = newPriceCurrency
		updatedPrices[currencyPair], _ = NewPriceCurrency(newPriceCurrency.Price)
		if helpers.NodeCfg.VerboseLog {
			log.Printf("Price Generator: %s Price Update: %.2f\n", currencyPair, newPriceCurrency.Price)
		}
	}
	// Send this batch of updates to the update channel
	if pg.updatesChan != nil && !UpdatesPaused {
		pg.updatesChan <- updatedPrices
	}
}

func (pg *PriceGeneratorApi) SetPricesUpdateChannel(updatesChan chan<- map[string]*PriceCurrency) {
	pg.updatesChan = updatesChan
}

func PauseUpdates() {
	UpdatesPaused = true
}

func ResumeUpdates() {
	UpdatesPaused = false
}

func (pg *PriceGeneratorApi) GenRandPricesForever(wg *sync.WaitGroup, min, max time.Duration, stopChan chan bool) {
	defer wg.Done()
	for {
		select {
		case stop := <-stopChan:
			if stop {
				return
			}
		default:
			pg.GenRandPrices(helpers.NodeCfg.UpdatesPercentChange)
			sleepTime := seedRand.Intn(int(max.Milliseconds()-min.Milliseconds()+1)) + int(min.Milliseconds())
			if helpers.NodeCfg.VerboseLog {
				fmt.Printf("Price Generator: Sleeping for %vms\n", sleepTime)
			}
			time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		}
	}
}

func (pg *PriceGeneratorApi) ManualPriceUpdate(updatedPrices map[string]*PriceCurrency) {
	pg.updatesChan <- updatedPrices
}
