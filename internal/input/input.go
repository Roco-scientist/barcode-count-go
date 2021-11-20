package input

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type SequenceFormat struct {
	Format_regex  regexp.Regexp
	format_string string
}

func (f *SequenceFormat) AddSearchRegex(format_file_path string) {
	var format_text string
	file, err := os.Open(format_file_path)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") {
			format_text += line
		}
	}
	digit_search := regexp.MustCompile(`\d+`)
	barcode_search := regexp.MustCompile(`(?i)(\{\d+\})|(\[\d+\])|(\(\d+\))|N+|[ATGC]+`)
	counted_barcode_num := 0
	var regex_string string
	for _, group := range barcode_search.FindAllString(format_text, -1) {
		var group_name string
		if strings.Contains(group, "[") {
			group_name = "sample"
		} else if strings.Contains(group, "{") {
			counted_barcode_num++
			group_name = fmt.Sprintf("counted_%v", counted_barcode_num)
		} else if strings.Contains(group, "(") {
			group_name = "random"
		}

		if len(group_name) != 0 {
			digits_string := digit_search.FindString(group)
			digits, _ := strconv.Atoi(digits_string)
			for i := 0; i < digits; i++ {
				f.format_string += "N"
			}
			regex_string += fmt.Sprintf("(?P<%v>[ATGCN]{%v})", group_name, digits_string)
		} else if strings.Contains(group, "N") {
			regex_string += fmt.Sprintf("[ATGCN]{%v}", len(group))
			f.format_string += group
		} else {
			regex_string += group
			f.format_string += group
		}

	}
	fmt.Println(f.format_string)
	f.Format_regex = *regexp.MustCompile(regex_string)
}

type SampleBarcodes struct {
	Conversion map[string]string
	Barcodes   []string
}

func NewSampleBarcodes(sample_file_path string) SampleBarcodes {
	file, err := os.Open(sample_file_path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var sample_barcodes SampleBarcodes

	scanner := bufio.NewScanner(file)
	scanner.Scan() // remove the header
	sample_barcodes.Conversion = make(map[string]string)
	for scanner.Scan() {
		row := strings.Split(scanner.Text(), ",")
		sample_barcodes.Conversion[row[0]] = row[1]
		sample_barcodes.Barcodes = append(sample_barcodes.Barcodes, row[0])
	}
	return sample_barcodes
}

type CountedBarcodes struct {
	Conversion []map[string]string
	Barcodes   [][]string
}

func NewCountedBarcodes(counted_bc_file_path string) CountedBarcodes {
	file, err := os.Open(counted_bc_file_path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var counted_barcodes CountedBarcodes

	scanner := bufio.NewScanner(file)
	scanner.Scan() // remove the header
	var barcode_nums []int
	var rows []string
	for scanner.Scan() {
		row_split := strings.Split(scanner.Text(), ",")
		barcode_num, _ := strconv.Atoi(row_split[2])
		barcode_nums = append(barcode_nums, barcode_num)
		rows = append(rows, row_split[0]+","+row_split[1]+","+row_split[2])
	}
	total_barcodes := max(barcode_nums)

	for i := 0; i < total_barcodes; i++ {
		counted_barcodes.Conversion = append(counted_barcodes.Conversion, make(map[string]string))
		counted_barcodes.Barcodes = append(counted_barcodes.Barcodes, make([]string, 0))
	}

	for _, row := range rows {
		row_split := strings.Split(row, ",")
		barcode_num, _ := strconv.Atoi(row_split[2])
		insert_num := barcode_num - 1
		counted_barcodes.Conversion[insert_num][row_split[0]] = row_split[1]
		counted_barcodes.Barcodes[insert_num] = append(counted_barcodes.Barcodes[insert_num], row_split[0])
	}
	return counted_barcodes
}

func max(int_slice []int) int {
	max_int := -10000000
	for _, integer := range int_slice {
		if integer > max_int {
			max_int = integer
		}
	}
	return max_int
}

func ReadFastq(fastq_path string, sequences chan string, wg *sync.WaitGroup) int {
	defer close(sequences)
	defer wg.Done()
	file, err := os.Open(fastq_path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	total_reads := 0
	line_num := 0
	for scanner.Scan() {
		line_num++
		switch line_num {
		case 2:
			total_reads++
			for len(sequences) > 10000 {
			}
			sequences <- scanner.Text()
			if total_reads%10000 == 0 {
				fmt.Printf("\rTotal reads: %v", total_reads)
			}
		case 4:
			line_num = 0
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\rTotal reads: %v\n", total_reads)
	return total_reads
}
