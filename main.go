package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

func main() {
	detail := flag.Bool("detail", false, "show per-event adjusted ranks in the no-podium table")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-detail] <competitionId>\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Fetches the public WCIF of a competition from the WCA API and prints")
		fmt.Fprintln(os.Stderr, "the best competitors without a podium and the best newcomers.")
		fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	w, err := fetchWCIF(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	s := Compute(w)
	if err := MarkNewcomers(s, w.ID, fetchPersonCompetitions); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	render(os.Stdout, s, *detail)
}

const newcomerEvent = "333"

func render(out io.Writer, s *Standings, detail bool) {
	fmt.Fprintln(out, s.CompetitionName)
	fmt.Fprintln(out, strings.Repeat("=", utf8.RuneCountInString(s.CompetitionName)))
	fmt.Fprintln(out)

	noPodium := s.WithoutPodium()
	headers := []string{"#", "Name", "WCA ID", "Sum"}
	if detail {
		headers = append(headers, s.EventIDs...)
	}
	sumKey := func(c *Competitor) int { return c.SumOfRanks }
	rows := tableRows(noPodium, Positions(noPodium, sumKey), func(c *Competitor) []string {
		cells := []string{c.Name, wcaIDCell(c), strconv.Itoa(c.SumOfRanks)}
		if detail {
			for _, ev := range s.EventIDs {
				cells = append(cells, strconv.Itoa(c.Adjusted[ev]))
			}
		}
		return cells
	})
	section(out, "Best without a podium (sum of ranks)", headers, rows)

	if !s.HasEvent(newcomerEvent) {
		fmt.Fprintln(out, "No 3x3x3 event; skipping newcomer ranking.")
		return
	}
	newcomers := s.Newcomers(newcomerEvent)
	key := func(c *Competitor) int { return c.Adjusted[newcomerEvent] }
	rows = tableRows(newcomers, Positions(newcomers, key), func(c *Competitor) []string {
		return []string{c.Name, strconv.Itoa(key(c))}
	})
	section(out, "Best newcomers (3x3x3 rank)", []string{"#", "Name", "3x3 rank"}, rows)
}

func wcaIDCell(c *Competitor) string {
	if c.WCAID == "" {
		return "newcomer"
	}
	return c.WCAID
}

func tableRows(list []*Competitor, pos []int, cells func(*Competitor) []string) [][]string {
	rows := make([][]string, len(list))
	for i, c := range list {
		p := strconv.Itoa(pos[i])
		if i > 0 && pos[i] == pos[i-1] {
			p = "="
		}
		rows[i] = append([]string{p}, cells(c)...)
	}
	return rows
}

func section(out io.Writer, title string, headers []string, rows [][]string) {
	fmt.Fprintln(out, title)
	fmt.Fprintln(out, strings.Repeat("-", len(title)))
	if len(rows) == 0 {
		fmt.Fprintln(out, "(none)")
		fmt.Fprintln(out)
		return
	}
	printTable(out, headers, rows)
	fmt.Fprintln(out)
}

func printTable(out io.Writer, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, r := range rows {
		for i, cell := range r {
			widths[i] = max(widths[i], utf8.RuneCountInString(cell))
		}
	}

	line := func(cells []string) {
		parts := make([]string, len(cells))
		for i, cell := range cells {
			pad := strings.Repeat(" ", widths[i]-utf8.RuneCountInString(cell))
			if isTextColumn(headers[i]) {
				parts[i] = cell + pad
			} else {
				parts[i] = pad + cell
			}
		}
		fmt.Fprintln(out, strings.TrimRight(strings.Join(parts, "  "), " "))
	}

	line(headers)
	sep := make([]string, len(headers))
	for i, w := range widths {
		sep[i] = strings.Repeat("-", w)
	}
	line(sep)
	for _, r := range rows {
		line(r)
	}
}

func isTextColumn(header string) bool {
	return header == "Name" || header == "WCA ID"
}
