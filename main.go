package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"

	"github.com/sausheong/petri"
)

var width int         // width of simulation grid
var interactions *int // how many cultural interactions
var coverage *float64 // how much of the grid is covered
var duration *int

// MASKARRAY is an array of masks used to replace the traits
var MASKARRAY []int = []int{0xFFFFF0, 0xFFFF0F, 0xFFF0FF, 0xFF0FFF, 0xF0FFFF, 0x0FFFFF}

var tick int // current simulation tick

// simulation data
var fdistances []string // average distance between features
var changes []string    // number of cultural changes
var uniques []string    // number of unique cultures

func main() {
	s := &CultureSim{}
	petri.Run(s)
}

func init() {
	width = *petri.Width
	interactions = flag.Int("n", 100, "number of interactions between cultures per simulation tick")
	coverage = flag.Float64("c", 1.0, "percentage of simulation grid that is populated with cultures")
	duration = flag.Int("d", 200, "the duration of the simulation")
	petri.Label = "Cultural Simulation"
}

type CultureSim struct {
	petri.Sim
}

func (sim *CultureSim) Exit() {
	saveData(fmt.Sprintf("n%d-w%d-c%1.1f", *interactions, width, *coverage))
}

func (sim *CultureSim) Init() {
	sim.Units = make([]petri.Cellular, width*width)
	n := 0
	for i := 1; i <= width; i++ {
		for j := 1; j <= width; j++ {
			p := rand.Float64()
			if p < *coverage {
				sim.Units[n] = sim.CreateCell(i, j, rand.Intn(0xFFFFFF), 0)
			} else {
				sim.Units[n] = sim.CreateCell(i, j, 0xFFFFFF, 0)
			}
			n++
		}
	}
	fdistances, changes, uniques = []string{"distance"}, []string{"change"}, []string{"unique"}
}

func (sim *CultureSim) Process() {
	var dist, chg, uniq int

	// if current tick is beyond simulation duration, save data and exit
	if tick > *duration {
		sim.Exit()
		os.Exit(1)
	}
	tick++

	for c := 0; c < *interactions; c++ {
		// randomly choose one cell
		r := rand.Intn(width * width)
		if sim.Units[r].RGB() != 0x0000 {
			// find all its neighbours
			neighbours := petri.FindNeighboursIndex(r)
			for _, neighbour := range neighbours {
				if sim.Units[neighbour].RGB() != 0x0000 {
					// cultural differences between the neighbour
					d := sim.diff(r, neighbour)
					// probability of a cultural exchange happening
					probability := 1 - float64(d)/96.0
					dp := rand.Float64()
					// cultural exchange happens
					if dp < probability {
						// randomly select one of the features
						i := rand.Intn(6)
						if d != 0 {
							var rp int
							// randomly select either trait to be replaced by the neighbour's
							if rand.Intn(1) == 0 {
								replacement := extract(sim.Units[r].RGB(), uint(i))
								rp = replace(sim.Units[neighbour].RGB(), replacement, uint(i))
							} else {
								replacement := extract(sim.Units[neighbour].RGB(), uint(i))
								rp = replace(sim.Units[r].RGB(), replacement, uint(i))
							}
							sim.Units[neighbour].SetRGB(rp)
							chg++
						}
					}

				}
			}
		}

		// calculate the average distance between all features and the number of unique cultures
		dist = sim.featureDistAvg()
		uniq = sim.similarCount()
	}
	fdistances = append(fdistances, strconv.Itoa(dist))
	changes = append(changes, strconv.Itoa(chg/width))
	uniques = append(uniques, strconv.Itoa(uniq))

	// clear screen first
	fmt.Print("\033[H\033[2J")
	fmt.Println("\nNumber of cultural interactions:", *interactions)
	fmt.Printf("\nSimulation coverage: %2.0f%%", *coverage*100)
	fmt.Printf("\nSimulation tick: %d/%d", tick, *duration)
	fmt.Println("\naverage distance between cultures:", dist,
		"\nnumber of unique cultures        :", uniq,
		"\nnumber of cultural exchanges     :", chg)
	fmt.Println("\nCtrl-c to quit simulation and save data.")
}

// total distance between traits for all features, between 2 cultures
func (sim *CultureSim) diff(a1, a2 int) int {
	var d int
	for i := 0; i < 5; i++ {
		d = d + traitDistance(sim.Units[a1].RGB(), sim.Units[a2].RGB(), uint(i))
	}
	return d
}

// average feature distance for the whole grid
func (sim *CultureSim) featureDistAvg() int {
	var count int
	var dist int
	for c := range sim.Units {
		neighbours := petri.FindNeighboursIndex(c)
		for _, neighbour := range neighbours {
			if sim.Units[neighbour].RGB() != 0x0000 {
				count++
				dist = dist + featureDistance(sim.Units[c].RGB(), sim.Units[neighbour].RGB())
			}
		}
	}
	return int(float64(dist/width) * (*coverage))
}

// distance between 2 features
func featureDistance(n1, n2 int) int {
	var features int = 0
	for i := 0; i < 5; i++ {
		f1, f2 := extract(n1, uint(i)), extract(n2, uint(i))
		if f1 == f2 {
			features++
		}
	}
	return 6 - features
}

// count unique colors
func (sim *CultureSim) similarCount() int {
	uniques := make(map[int]int)
	for _, c := range sim.Units {
		uniques[c.RGB()] = c.RGB()
	}
	return len(uniques)
}

// find the distance of 2 numbers at position pos
func traitDistance(n1, n2 int, pos uint) int {
	d := extract(n1, pos) - extract(n2, pos)
	if d < 0 {
		return d * -1
	}
	return d
}

// extract trait for 1 feature
func extract(n int, pos uint) int {
	return (n >> (4 * pos)) & 0x00000F
}

// replace the trait in 1 feature
func replace(n, replacement int, pos uint) int {
	i1 := n & MASKARRAY[pos]
	mask2 := replacement << (4 * pos)
	return (i1 ^ mask2)
}

// save simulation data
func saveData(name string) {
	// simulation data
	data := [][]string{
		fdistances, // average feature distance
		changes,    // number of changes
		uniques}    // number of unique cultures
	csvfile, err := os.Create(fmt.Sprintf("data/log-%s.csv", name))
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	csvwriter := csv.NewWriter(csvfile)

	for _, line := range data {
		_ = csvwriter.Write([]string(line))
	}
	csvwriter.Flush()
	csvfile.Close()
	fmt.Printf("\nSimulation data saved in data/log-%s.csv saved.\n", name)
}
