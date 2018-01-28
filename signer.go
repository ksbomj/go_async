package main

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"
)

/*
ExecutePipeline - function conveyor
*/
func ExecutePipeline(freeFlowJobs ...job) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	chanIn := make(chan interface{}, 1)
	chanOut := make(chan interface{}, 1)

	for _, freeJob := range freeFlowJobs {
		chanIn = chanOut
		chanOut = make(chan interface{}, 1)
		wg.Add(1)
		go func(someJob job, in, out chan interface{}) {
			someJob(in, out)
			wg.Done()
			close(out)
		}(freeJob, chanIn, chanOut)
	}
}

func SingleHash(in, out chan interface{}) {
	fmt.Println("Single")
	// SingleHash считает значение crc32(data)+"~"+crc32(md5(data)) ( конкатенация двух строк через ~),
	// где data - то что пришло на вход (по сути - числа из первой функции)
	wgMain := sync.WaitGroup{}
LOOP:
	for {
		time.Sleep(time.Millisecond * 30)
		select {
		case dataRaw := <-in:
			data, ok := dataRaw.(int)
			if !ok {
				fmt.Println("Single: cant convert result data to string")
				break LOOP
			}
			wgMain.Add(1)
			go func(someData int) {
				fmt.Println(someData)
				strData := strconv.Itoa(someData)
				var strCrc32 string
				var strMd5 string
				wg := sync.WaitGroup{}
				wg.Add(1)
				go func(temp string) {
					strCrc32 = DataSignerCrc32(temp)
					wg.Done()
				}(strData)
				wg.Add(1)
				go func(temp string) {
					strMd5 = DataSignerCrc32(DataSignerMd5(temp))
					wg.Done()
				}(strData)
				wg.Wait()
				out <- strCrc32 + "~" + strMd5
				wgMain.Done()
			}(data)
			continue
		}
	}
	fmt.Println("Single wait...")
	wgMain.Wait()
	fmt.Println("Single done")
	return
}

func MultiHash(in, out chan interface{}) {
	fmt.Println("Multi")
	// MultiHash считает значение crc32(th+data)) (конкатенация цифры, приведённой к строке и строки),
	// где th=0..5 ( т.е. 6 хешей на каждое входящее значение ),
	// потом берёт конкатенацию результатов в порядке расчета (0..5),
	// где data - то что пришло на вход (и ушло на выход из SingleHash)
	// var totalOperations int32
	wgMain := &sync.WaitGroup{}
LOOP:
	for {
		select {
		case dataRaw := <-in:
			data, ok := dataRaw.(string)
			if !ok {
				fmt.Println("Multi :cant convert result data to string")
				break LOOP
			}
			wgMain.Add(1)
			go func(s string) {
				wg := &sync.WaitGroup{}
				var arrTemp [6]string
				temp := ""
				for i := 0; i < 6; i++ {
					wg.Add(1)
					go func(j int, someData string) {
						arrTemp[j] = DataSignerCrc32(strconv.Itoa(j) + someData)
						wg.Done()
					}(i, data)
				}
				wg.Wait()
				for _, item := range arrTemp {
					temp += item
				}
				fmt.Println("temp:", temp)
				out <- temp
				wgMain.Done()
			}(data)
			continue
		}
	}
	fmt.Println("Multi wait...")
	wgMain.Wait()
	fmt.Println("Multi done")
	return
}

func CombineResults(in, out chan interface{}) {
	// CombineResults получает все результаты, сортирует (https://golang.org/pkg/sort/),
	// объединяет отсортированный результат через _ (символ подчеркивания) в одну строку
	fmt.Println("Combine")
	var result []string
	var temp string
	// var totalOperations int32
LOOP:
	for {
		select {
		case dataRaw := <-in:
			data, ok := dataRaw.(string)
			if !ok {
				fmt.Println("Combine: cant convert result data to string")
				break LOOP
			}
			result = append(result, data)
		}
	}
	fmt.Println("End append")
	sort.Strings(result)
	temp = ""
	for i, str := range result {
		if i != len(result)-1 {
			temp += str + "_"
		} else {
			temp += str
		}
	}
	fmt.Println(temp)
	out <- temp
	return
}
