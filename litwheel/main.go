package main

// not highly efficient method to parse raidtool information
// creates raidtooltmp.txt, raidtool.txt files in the local directory
import (
	"encoding/csv"
	"fmt"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	// command line check >>
	if len(os.Args) < 5 {
		fmt.Println("LITwheel \n \n ================== \n A program to run TWIX backups \n ================= \n \n"+
			"Usage: \n litwheel -hashlog=HASHFILE.txt -key=SSH_KEY -user=USER -address=IP_ADDRESS:/PATH -debug=LOOP_NUMBER(default 0)  \n" +
			"\nREQUIRED: \n" +
			" HASHFILE.txt - can be an existing list of hashes, or this tool will create a new file with the given name. \n" +
			" SSH_KEY - key for user stored in local directory \n" +
			" USER - username to access IP address \n" +
			" IP_ADDRESS:/PATH - IP address and target path for storage \n" +
			"\nOPTIONAL: \n" +
			" LOOP_NUMBER - number of loops to run for debugging")
		//"\n all : OPTIONAL- will force transfer of all data on the RAID, otherwise will check Performing Physician field for the following format:" +
		//" \"PERF PHYS NAME, [A-Z]{4,5}[0-9]{4,6}-[A-Z0-9]{4,10} \" (i.e. NHLBI1234-A0001)- where the comma is the separator key")
		os.Exit(0)
	}


	// *********************************************************************
	// PARSE COMMAND LINE INPUTS
	// *********************************************************************

  // read text file from raidtool dump
	raidfilePtr := flag.String("hashlog", "hashlog.txt", "the hashlog")
	userkeyPtr := flag.String("key", " ", "user ssh key")
	usernamePtr := flag.String("user", "meduser", "username")
	storageaddressPtr := flag.String("address", "192.168.2.5:/data/LITwheel/", "storage destination address")
	debugTickPtr := flag.Int("debug", 0, "number of debug ticks")
	flag.Parse()

	fmt.Println("hashlog:", *raidfilePtr)
	fmt.Println("user key text file:", *userkeyPtr)
	fmt.Println("user:", *usernamePtr)
	fmt.Println("storage address and path:", *storageaddressPtr)
	fmt.Println("ticks:", *debugTickPtr)

	// check if hash record exists >>
	_, err := os.Open(*raidfilePtr)
	if err != nil {
		fmt.Printf("This file does not exist, creating %v\n", *raidfilePtr)
		_, err = os.Create(*raidfilePtr)
		if err != nil {
			panic(err)
		}
	}
	// check if hash record exists <<

	// raidtool dump >>
	// debug //	fmt.Println("Raidtool dump") // debug //
	cmd := exec.Command("cmd.exe", "/C", "raidtool -d -a mars -p 8010 > raidtool.txt")
	// offline debug // cmd := exec.Command("cmd.exe", "/C", "RR_rt_print.exe > rt_temp.txt") // offline debug //
	//stdout, err := cmd.Output()
	_, err = cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	// raidtool dump <<

	// load raidtool dump >>
	// debug //	fmt.Println("Raidtool read") // debug //
	rtFile, err2 := ioutil.ReadFile("raidtool.txt")
	if err2 != nil {
		log.Fatal(err2)
	}
	// load raidtool dump <<

	// raidtool header print >>
	rt_string := string(rtFile[:])
	idx := strings.Index(rt_string, "FileID")
	rt_head := rt_string[:idx]
	headSlice := strings.Split(rt_head, " ")
	numFiles, _ := strconv.Atoi(headSlice[35]) // empirically consistent
	fileIDs := make([]string, numFiles+20)     // padding to avoid 'panic'
	fmt.Println("fileID size", len(fileIDs), "rt_head: \n", rt_head)
	// raidtool header print <<

	// Attempt to find measurement IDs using csv (tab delimiting doesn't quite work)
	idx = strings.Index(rt_string, "(fileID)")
	rt_body := rt_string[idx+len("(fileID)"):]
	r := csv.NewReader(strings.NewReader(rt_body))
	r.Comma = '\t' // ? is this reduntant?

	// loop through raidtool dump >>

	if *debugTickPtr !=0 {
		fmt.Printf("Limited operation is in effect. Will run %d loops.\n", *debugTickPtr)
	}

	// *************************

	// *********************************************************************
	// THE START OF THE FIRST LOOP (CREATING isNotToBeCopiedArray)
	// *********************************************************************

		isNotToBeCopiedArray := make([]int, 0)    // figure out how to initialize this stuff (arrays)
		fileNameStrArray := make([]string, 0)
		fileIDArray := make([]string, 0)

		raidLoopCounter := 0
		for {
			fmt.Printf("**************\n")
			// debug //		fmt.Println("Reading CSV") // debug //


			if (raidLoopCounter + 1 > *debugTickPtr) && (*debugTickPtr > 0) {

				break
			} // limit how much of the RAID is processed for testing */





			record, err := r.Read()
			if err == io.EOF {
				break
			}

			if err != nil {
				log.Fatal(err)
			}

			reg, err2 := regexp.Compile("[^0-9]+")
			if err2 != nil {
				log.Fatal(err2)
			}

			raidLoopCounter += 1

			newRaidLine := record[0]
			// debug // fmt.Println("New line read") // fmt.Println(newRaidLine + "\n++++++++++++++\n") // debug //
			if len(record[0]) < 100 { // end of file catch
				break
			}

	// debug //
			// fmt.Println("New line read") //
			 // debug //
			newRaidLineSplit := strings.SplitAfterN(newRaidLine, " ", 500)

			elementStr := "tmpstr"
			isNotToBeCopied := 0
			elementNumber := 0
			i := 0

			// index := 0
			protNameFlag := 0

			fileID := "und" // undefined
			MeasID := "und"
			fileNameStr := "und"
			dateStr := "und"
			timeStr := "und"




			for elementNumber < 9 {



				elementStr = newRaidLineSplit[i]
				// fmt.Printf(elementStr + "\n")

				elementStr = strings.Replace(elementStr, " ", "", -1)

				if len(elementStr) > 0 {
					// d = unicode.IsNumber(rune(elementStr[0]))
					// fmt.Println(">>" +elementStr)

					//		fmt.Println(IsLetter(string(elementStr[0])))
					elementNumber += 1

					if elementNumber == 1 {
						fileID = elementStr
					} else if elementNumber == 2 {
						if len(elementStr) > 5 { // retrorecon jobs have 7-digit FID's, no need to download these duplicates.
							fmt.Println("retrorecon")
							isNotToBeCopied = 1
						} else {
							MeasID = strings.Repeat("0", 5-len(elementStr)) + elementStr
						}
					} else if elementNumber == 3 { // this should be tidied
						fileNameStr = elementStr
						if len(elementStr) > 2 {
							//fmt.Println(elementStr+" strcmp: %d", strings.Compare(elementStr[0:3], "Adj"))

							if elementStr[0:3] == "Adj" {
								isNotToBeCopied = 1 // borrowing retrorecon flag to not copy adjustment scans
								fmt.Println("adj")
							}
						}
					} else if elementNumber > 3 && elementNumber < 7 { // sift through possible spaces in the filename

						// currently, xxxxxx for PatName when using anonymized raid
						if elementStr != "xxxxxx" && protNameFlag == 0 {
							elementNumber -= 1
							fileNameStr = fileNameStr + "_" + elementStr
						} else if elementStr == "xxxxxx" && protNameFlag == 0 {
						  protNameFlag = 1
						}
					} else if elementNumber == 8 {
						//fmt.Println(elementStr)
						date1 := elementStr
						dateStr = date1[6:10] + date1[3:5] + date1[0:2]
						//fmt.Println("This is the date str: " + dateStr)
						/*fmt.Println(date1[6:10])
						fmt.Println(date1[3:5])
						fmt.Println(date1[0:2])
						fmt.Println(len(date1)) */
					} else if elementNumber == 9 {
						//fmt.Println(elementStr)
						time1 := elementStr
						timeStr = reg.ReplaceAllString(time1, "")
						//			fmt.Println("This is the time str: " + timeStr)
					}

				} else {
				}
				i += 1
			}

			// stash everything into arrays (fileID, measID, ...)

			isNotToBeCopiedArray=append(isNotToBeCopiedArray, isNotToBeCopied)
			fileIDArray=append(fileIDArray,fileID)
			// measIDArray=append(measIDArray,MeasID)


			//	fileID := reg.ReplaceAllString(newRaidLine[0:10], "") // [0:10]-12 is affected by retrorecon, 8 is still safe with len(FILEID)=4
			fileNameStr = dateStr + "_" + timeStr + "_" + "meas_" + "MID" + MeasID + "_FID" + strings.Repeat("0", 5-len(fileID)) + fileID + "_" + fileNameStr + ".dat" // get for list making purposes
			//	fmt.Println("FILE ID: " + fileID) // debug //
			fmt.Println(fileNameStr)
			fileNameStrArray=append(fileNameStrArray, fileNameStr)


		} 	// loop through raidtool dump << (raidLoop end)


		// *********************************************************************
		// THE START OF THE SECOND LOOP (COPYING DATA BASED ON isNotToBeCopiedArray)
		// *********************************************************************





		// start loop for len(raidLoopCounter)

hashlog, err := ioutil.ReadFile(*raidfilePtr)



		for j := 0; j < raidLoopCounter; j++ {
			fmt.Printf("***************\n")

			index := raidLoopCounter - j - 1 // number of files = raidLoopCounter



			if isNotToBeCopiedArray[index] == 0 {  // isNotToBeCopiedArray = all isNotToBeCopied's

				fmt.Printf("raidtool -m " + fileIDArray[index] + " -o " + fileNameStrArray[index] + " -a mars -p 8010 \n")
				fmt.Printf("scp " + fileNameStrArray[index] + " -i " + *userkeyPtr + " " + *usernamePtr + "@" + *storageaddressPtr + "\n") //change to inputs for target address and key
				fmt.Printf("rm " + fileNameStrArray[index] + " \n")


				// Suitable for transfer - now check if hash exists locally >>

				// debug //fmt.Println("raidtool -h "+fileID+" -o raidtooltmp.txt -a mars -p 8010") // offline debug //
				cmd := exec.Command("cmd.exe", "/C", "raidtool -h "+fileIDArray[index]+" -o raidtooltmp.txt -a mars -p 8010")
				_, err = cmd.Output()
				if err != nil {
					log.Fatal(err)
				}

				cmd = exec.Command("cmd.exe", "/C", "hdrsignature raidtooltmp.txt")
				stdout, err := cmd.Output()
				if err != nil {
					panic(err)
				}







				hdrHash := string(stdout[:])

				// check if hash exists >>


				if strings.Contains(string(hashlog), hdrHash) {

					fmt.Println("file ID " + fileIDArray[index] + " : Hash exists, no need to transfer") // debug //

				} else {

					fmt.Println("file ID " + fileIDArray[index] + " : No hash, transferring and appending to log. ***")
					// @ LIT wheel - Data transfer >>

					// download data : -D > dependent measurements
					// debug // fmt.Printf("raidtool -m " + fileID + " -o " + fileNameStr + " -a mars -p 8010 -D \n")

					cmd = exec.Command("cmd.exe", "/C", "raidtool -m "+fileIDArray[index]+" -o "+fileNameStrArray[index]+" -a mars -p 8010 -D")

					_, err = cmd.Output()
					if err != nil {
						log.Fatal(err)
					}

					// transfer data & remove host copy
					cmd = exec.Command("cmd.exe", "/C", "scp " + fileNameStrArray[index] + " -i " + *userkeyPtr + " " + *usernamePtr + "@" + *storageaddressPtr)

					_, err = cmd.Output()
					if err != nil {
						log.Fatal(err)
					}

					// debug //
					//		fmt.Printf("rm " + fileNameStr + " \n")
					cmd = exec.Command("cmd.exe", "/C", "rm "+fileNameStrArray[index])

					_, err = cmd.Output()
					if err != nil {
						log.Fatal(err)
					}
					// @ LIT wheel - Data transfer <<

					// append hash >>

					f, err := os.OpenFile(*raidfilePtr, os.O_APPEND, 0660)
					if err != nil {
						panic(err)
					}

					// debug //				n3, err := f.WriteString(hdrHash) // debug //
					_, err = f.WriteString(hdrHash)
					if err != nil {
						panic(err)
					}
					// debug //		fmt.Printf("wrote %d bytes\n", n3) // debug //
					f.Sync()

					// append hash <<

				} // check if hash exists <<

			}

		}




}
