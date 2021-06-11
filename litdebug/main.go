package main

// not highly efficient method to parse raidtool information
// Steps:
// 	- Query RAID for files and pipe to local file (can be used for logging)
// 	- Run through File IDs and determine filetype (meas,adj,retrorecon)
// 	- stash files to-be-copied in an array
// 	- Run through the data (from oldest to newest) and hash the header
// 	- if hash exists in local reference file, data has already been copied
//	- data will be copied using scp and ssh keys
// Notes:
// 	- creates raidtooltmp.txt, raidtool.txt files in the local directory
// 	- Retro-recon and "Adj" scans will not be copied in this version.
// Usage:
// 	Standard call:
// 	- litwheel -user=<username> -key=<sshkey> -address=<storage_ip:storage_dir> 
// 	Debugging a stored RAID file:
// 	- litwheel -file=<RAIDfile> -debug=<NUM_steps> -user=<username> -key=<sshkey> -address=<storage_ip:storage_dir> 

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	//	"os"
	//	"os/exec"
	"regexp"
	// "strconv"
	"strings"
	// "unicode"
)

func main() {

	fmt.Println("^^^^^^^^^^^^^^^^^^^\nSTART OF FUNCTION")

	// *********************************************************************
	// PARSE COMMAND LINE INPUTS
	// *********************************************************************

  	// read text file from raidtool dump 
	raidfilePtr := flag.String("file", "raidtool.txt", "a raidfile txt")
	userkeyPtr := flag.String("key", "tempkey", "user ssh key")
	usernamePtr := flag.String("user", "meduser", "username for server")
	storageaddressPtr := flag.String("address", "XXX.XXX.X.X:/target_dir/", "storage destination address")
	debugTickPtr := flag.Int("debug", 0, "number of debug ticks")
	flag.Parse()

	// fmt.Println("raidfile text:", *raidfilePtr)
	// fmt.Println("user key text file:", *userkeyPtr)
	// fmt.Println("user:", *usernamePtr)
	// fmt.Println("ticks:", *debugTickPtr)

	// load raidtool dump >>
	// debug //	fmt.Println("Raidtool read") // debug //
	rtFile, err2 := ioutil.ReadFile(*raidfilePtr)
	if err2 != nil {
		log.Fatal(err2)
	}

	// load raidtool dump <<

	
	// *********************************************************************
	// PRINT RAIDTOOL HEADER
	// *********************************************************************

	rt_string := string(rtFile[:])
	idx := strings.Index(rt_string, "FileID")
	rt_head := rt_string[:idx]
	// headSlice := strings.Split(rt_head, " ")
	// numFiles, _ := strconv.Atoi(headSlice[35]) // empirically consistent
	// fileIDs := make([]string, numFiles+20)     // padding to avoid 'panic'
	fmt.Println("^^^^^^^^^^^^^^^^^^^\nSTART OF HEADER", rt_head)
	// raidtool header print <<

	// Attempt to find measurement IDs using csv (tab delimiting doesn't quite work)
	idx = strings.Index(rt_string, "(fileID)")
	rt_body := rt_string[idx+len("(fileID)"):]
	r := csv.NewReader(strings.NewReader(rt_body))
	r.Comma = '\t'
	// loop through raidtool dump >>

	if *debugTickPtr !=0 {
		fmt.Printf("Limited operation is in effect. Will run %d loops.\n", *debugTickPtr)
	}


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
			// VE11A-C RAID compatible, picking out first 9 elements of RAID columns.

			elementStr = newRaidLineSplit[i]
			// fmt.Printf(elementStr + "\n")

			elementStr = strings.Replace(elementStr, " ", "", -1)

			if len(elementStr) > 0 {
				
				elementNumber += 1

				if elementNumber == 1 {
					// FILE ID
					fileID = elementStr
					
				} else if elementNumber == 2 {
					// MEAS ID
					if len(elementStr) > 5 { // retrorecon jobs have 7-digit FID's, no need to download these duplicates.
						fmt.Println("retrorecon")
						isNotToBeCopied = 1
					} else {
						MeasID = strings.Repeat("0", 5-len(elementStr)) + elementStr
					}
					
				} else if elementNumber == 3 { // this should be tidied
					// FILE NAME
					fileNameStr = elementStr
					if len(elementStr) > 2 {
						//fmt.Println(elementStr+" strcmp: %d", strings.Compare(elementStr[0:3], "Adj"))

						if elementStr[0:3] == "Adj" {
							isNotToBeCopied = 1 // borrowing retrorecon flag to not copy adjustment scans
							fmt.Println("adj")
						}
					}
					
				} else if elementNumber > 3 && elementNumber < 7 { // sift through possible spaces in the filename
					// FILE NAME (cont) 
					// Currently, xxxxxx for PatName when using anonymized raid
					// use this to identify end-of-file-name
					
					if elementStr != "xxxxxx" && protNameFlag == 0 {
						elementNumber -= 1
						fileNameStr = fileNameStr + "_" + elementStr
					} else if elementStr == "xxxxxx" && protNameFlag == 0 {
					  protNameFlag = 1
					}
					
				} else if elementNumber == 8 {
					// DATE CREATED
					date1 := elementStr
					dateStr = date1[6:10] + date1[3:5] + date1[0:2]
					
				} else if elementNumber == 9 {
					// TIME CREATED 
					time1 := elementStr
					timeStr = reg.ReplaceAllString(time1, "")
					
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
	// SECOND LOOP (COPYING DATA BASED ON isNotToBeCopiedArray)
	// *********************************************************************

	for j := 0; j < raidLoopCounter; j++ {
		index := raidLoopCounter - j - 1 // number of files = raidLoopCounter



		if isNotToBeCopiedArray[index] == 0 {  // isNotToBeCopiedArray = all isNotToBeCopied's

			// get header >> turn header into hash >> cross reference hash

			fmt.Printf("raidtool -m " + fileIDArray[index] + " -o " + fileNameStrArray[index] + " -a mars -p 8010 \n")
			fmt.Printf("scp " + fileNameStrArray[index] + " -i " + *userkeyPtr + " " + *usernamePtr + "@" + *storageaddressPtr + "\n") //change to inputs for target address and key
			fmt.Printf("rm " + fileNameStrArray[index] + " \n")
			// target format: meas_MID00000_FID00000_NAME.dat
		}
	}


}
