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
	chanIn := make(chan interface{}, 100)
	chanOut := make(chan interface{}, 100)
	for _, freeJob := range freeFlowJobs {
		chanIn = chanOut
		chanOut = make(chan interface{}, 100)
		wg.Add(1)
		go func(someJob job, in, out chan interface{}) {
			someJob(in, out)
			// freeJob(chanIn, chanOut)
			wg.Done()
		}(freeJob, chanIn, chanOut)
		// time.Sleep(time.Millisecond * 10)
		// runtime.Gosched()
	}
	wg.Wait()
	fmt.Println("End pipeline")
}

/*
SingleHash ...
*/
func SingleHash(in, out chan interface{}) {
	fmt.Println("Single")
	// SingleHash считает значение crc32(data)+"~"+crc32(md5(data)) ( конкатенация двух строк через ~),
	// где data - то что пришло на вход (по сути - числа из первой функции)
	isDoWork := false
	for {
		// fmt.Println("Single: HELP!")
		time.Sleep(time.Millisecond * 20)
		select {
		case dataRaw := <-in:
			data, ok := dataRaw.(int)
			if !ok {
				fmt.Println(dataRaw)
				fmt.Println("cant convert result data to string")
			}
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
					// strCrc32 = DataSignerCrc32(strData)
				}(strData)
				wg.Add(1)
				go func(temp string) {
					strMd5 = DataSignerCrc32(DataSignerMd5(temp))
					wg.Done()
					// strMd5 = DataSignerCrc32(DataSignerMd5(strData))
				}(strData)
				wg.Wait()
				out <- strCrc32 + "~" + strMd5
				isDoWork = true
			}(data)
		default:
			if isDoWork {
				fmt.Println("goodbye Single")
				return
			}
			continue
		}
	}
}

func MultiHash(in, out chan interface{}) {
	fmt.Println("Multi")
	// MultiHash считает значение crc32(th+data)) (конкатенация цифры, приведённой к строке и строки),
	// где th=0..5 ( т.е. 6 хешей на каждое входящее значение ),
	// потом берёт конкатенацию результатов в порядке расчета (0..5),
	// где data - то что пришло на вход (и ушло на выход из SingleHash)
	isDoWork := false
	for {
		// fmt.Println("Multi: HELP!")
		select {
		case dataRaw := <-in:
			data, ok := dataRaw.(string)
			if !ok {
				fmt.Println("cant convert result data to string")
				return
			}
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
				fmt.Println(temp)
				out <- temp
				isDoWork = true
			}(data)
		default:
			if isDoWork {
				fmt.Println("goodbye Multi")
				return
			}
		}
	}
	// }
	// for dataRaw := range in {
	// 	data, ok := dataRaw.(string)
	// 	if !ok {
	// 		fmt.Println("cant convert result data to string")
	// 	}
	// 	go func(s string) {
	// 		wg := &sync.WaitGroup{}
	// 		var arrTemp [6]string
	// 		temp := ""
	// 		for i := 0; i < 6; i++ {
	// 			wg.Add(1)
	// 			go func(j int, someData string) {
	// 				arrTemp[j] = DataSignerCrc32(strconv.Itoa(j) + someData)
	// 				wg.Done()
	// 			}(i, data)
	// 		}
	// 		wg.Wait()
	// 		for _, item := range arrTemp {
	// 			temp += item
	// 		}
	// 		fmt.Println(temp)
	// 		out <- temp
	// 	}(data)
	// }
}

func CombineResults(in, out chan interface{}) {
	// CombineResults получает все результаты, сортирует (https://golang.org/pkg/sort/),
	// объединяет отсортированный результат через _ (символ подчеркивания) в одну строку
	fmt.Println("Combine")
	var result []string
	var temp string
	isDoWork := false
LOOP:
	for {
		select {
		case dataRaw := <-in:
			fmt.Println("Combine: HELP!")
			data, ok := dataRaw.(string)
			if !ok {
				fmt.Println("cant convert result data to string")
			}
			fmt.Println(data)
			result = append(result, data)
			isDoWork = true
		default:
			if isDoWork {
				fmt.Println("goodbye Multi")
				break LOOP
			}
		}
	}
	// for dataRaw := range in {
	// 	fmt.Println("Combine: HELP!")
	// 	data, ok := dataRaw.(string)
	// 	if !ok {
	// 		fmt.Println("cant convert result data to string")
	// 	}
	// 	fmt.Println(data)
	// 	result = append(result, data)
	// }
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
	fmt.Println("Out result: ", temp)
	// fmt.Println(temp)
	out <- temp
	return
}

// crc32 считается через функцию DataSignerCrc32
// md5 считается через DataSignerMd5
// В чем подвох:

// DataSignerMd5 может одновременно вызываться только 1 раз, считается 10 мс.
// Если одновременно запустится несколько - будет перегрев на 1 сек
// DataSignerCrc32, считается 1 сек
// На все расчеты у нас 3 сек.
// Если делать в лоб, линейно - для 7 элементов это займёт почти 57 секунд, следовательно надо это как-то распараллелить
