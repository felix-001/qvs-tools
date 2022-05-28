package main

import (
	"io/ioutil"
	"log"
	"strings"
)

func partition(arr []string, low, high int) ([]string, int) {
	pivot := arr[high]
	i := low
	for j := low; j < high; j++ {
		//		if arr[j] < pivot {
		if strings.Compare(arr[j], pivot) == -1 {
			arr[i], arr[j] = arr[j], arr[i]
			i++
		}
	}
	arr[i], arr[high] = arr[high], arr[i]
	return arr, i
}

func quickSort(arr []string, low, high int) []string {
	if low < high {
		var p int
		arr, p = partition(arr, low, high)
		arr = quickSort(arr, low, p-1)
		arr = quickSort(arr, p+1, high)
	}
	return arr
}

func main() {
	b, err := ioutil.ReadFile("/tmp/pdr.log")
	if err != nil {
		log.Println("read fail", "/tmp/pdr.log", err)
		return
	}
	logs := strings.Split(string(b), "\n")
	validLogs := []string{}
	for _, log := range logs {
		if len(log) > 0 && log[0] == '[' {
			validLogs = append(validLogs, log)
		}
	}
	res := quickSort(validLogs, 0, len(validLogs)-1)
	txt := ""
	for _, log := range res {
		txt += log + "\n"
	}
	err = ioutil.WriteFile("/tmp/pdr2.log", []byte(txt), 0644)
	if err != nil {
		log.Println(err)
		return
	}
}
