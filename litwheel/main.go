package main

// command-line function to send raw Siemens MRI data to a target directory
// Steps:
// 	- Query RAID for files and pipe to local file (can be used for logging)
// 	- Run through File IDs and determine filetype (meas,adj,retrorecon)
// 	- stash files to-be-copied in an array
// 	- Run through the data (from oldest to newest) and hash the header
// 	- if hash exists in local reference file, data has already been copied
//	- data will be copied using scp and ssh keys
// Notes:
// 	- OVERHEAD: creates raidtooltmp.txt, raidtool.txt files in the local directory
// 	- OVERHEAD/TODO: Hashlog file will currently continue to grow
// 	- OVERHEAD/TODO: Files will not be sorted at target, it will just be one giant folder unless you tidy it up.
//  - DEPENDENCY: Requires hdrsignature compiled and in the same directory as litwheel. 
// 	- BEHAVIOUR: Retro-recon and "Adj" scans will not be copied in this version.
// Usage:
// 	Standard call:
// 	- litwheel -user=<username> -key=<sshkey> -address=<storage_ip:storage_dir> 
// 	Debugging a stored RAID file:
// 	- litwheel -file=<RAIDfile> -debug=<NUM_steps> -user=<username> -key=<sshkey> -address=<storage_ip:storage_dir> 

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
	// *********************************************************************
	// COMMAND LINE HELP
	// *********************************************************************
	if len(os.Args) < 5 {
		fmt.Println("\n ================== \n LITwheel \n ================= \n\n"+
			"Description: \n A program to run TWIX backups. \n\n" +
			"Usage: \n litwheel -hashlog=HASHFILE.txt -key=SSH_KEY -user=USER -address=IP_ADDRESS:/PATH -debug=LOOP_NUMBER(default 0)  \n" +
			"\nREQUIRED: \n" +
			" HASHFILE.txt - can be an existing list of hashes, or this tool will create a new file with the given name. \n" +
			" SSH_KEY - key for user stored in local directory \n" +
			" USER - username to access IP address \n" +
			" IP_ADDRESS:/PATH - IP address and target path for storage \n" +
			"\nOPTIONAL: \n" +
			" LOOP_NUMBER - number of loops to run for debugging")
		
		// POTENTIAL ALTERNATIVE USAGE: the PERF PHYS NAME field name can be accessed in anonymized dats, so a key in this field could be used to flag for transfer. 
		// [A-Z]{4,5}[0-9]{4,6}-[A-Z0-9]{4,10} \" (i.e. NHLBI1234-A0001)")
		os.Exit(0)
	}


	// *********************************************************************
	// PARSE COMMAND LINE INPUTS
	// *********************************************************************

	raidfilePtr := flag.String("hashlog", "hashlog.txt", "the hashlog")
	userkeyPtr := flag.String("key", " ", "user ssh key")
	usernamePtr := flag.String("user", "meduser", "username")
	storageaddressPtr := flag.String("address", "XXX.XXX.X.X:/target_dir/", "storage destination address")
	debugTickPtr := flag.Int("debug", 0, "number of debug ticks")
	flag.Parse()

	// debug // 
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

	// *********************************************************************
	// READ RAID & DUMP TO FILE
	// *********************************************************************

	// debug //	fmt.Println("Raidtool dump") // debug //
	cmd := exec.Command("cmd.exe", "/C", "raidtool -d -a mars -p 8010 > raidtool.txt")
	// offline debug // cmd := exec.Command("cmd.exe", "/C", "RR_rt_print.exe > rt_temp.txt") // offline debug //
	//stdout, err := cmd.Output()
	_, err = cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	
	rtFile, err2 := ioutil.ReadFile("raidtool.txt")
	if err2 != nil {
		log.Fatal(err2)
	}
	
	// *********************************************************************
	// raidtool header print
	// *********************************************************************
	
	rt_string := string(rtFile[:])
	idx := strings.Index(rt_string, "FileID")
	rt_head := rt_string[:idx]
	headSlice := strings.Split(rt_head, " ")
	numFiles, _ := strconv.Atoi(headSlice[35]) // empirically consistent
	fileIDs := make([]string, numFiles+20)     // padding to avoid 'panic'
	fmt.Println("fileID size", len(fileIDs), "rt_head: \n", rt_head)

	// Attempt to find measurement IDs using csv (tab delimiting doesn't quite work)
	idx = strings.Index(rt_string, "(fileID)")
	rt_body := rt_string[idx+len("(fileID)"):]
	r := csv.NewReader(strings.NewReader(rt_body))
	r.Comma = '\t' // ? is this reduntant?

	// *********************************************************************
	// FIRST LOOP - STASH FILEIDS 
	// *********************************************************************

	if *debugTickPtr !=0 {
		fmt.Printf("Limited operation is in effect. Will run %d loops.\n", *debugTickPtr)
	}

	// FileID arrays and loop-counting vars
	isNotToBeCopiedArray := make([]int, 0)    
	fileNameStrArray := make([]string, 0)
	fileIDArray := make([]string, 0)

	raidLoopCounter := 0 // 
	
	for {

		// limit how much of the RAID is processed for testing
		if (raidLoopCounter + 1 > *debugTickPtr) && (*debugTickPtr > 0) {

			break
		} 

		// Read until end-of-file
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		raidLoopCounter += 1

		if err != nil {
			log.Fatal(err)
		}

		// Parse line
		reg, err2 := regexp.Compile("[^0-9]+")
		if err2 != nil {
			log.Fatal(err2)
		}

		newRaidLine := record[0]

		if len(record[0]) < 100 { // end of file catch
			break
		}

		newRaidLineSplit := strings.SplitAfterN(newRaidLine, " ", 500) // this delimits by spaces, so the next block of code is dealing with this mess..

		// initialise vars for processing
		elementStr := "tmpstr"
		isNotToBeCopied := 0
		elementNumber := 0
		i := 0
		protNameFlag := 0
		fileID := "und" // undefined
		MeasID := "und"
		fileNameStr := "und"
		dateStr := "und"
		timeStr := "und"

		// Parse the TWIX columns we care about:
		// (FileID | MeasID | ProtName | CreationTime)
		// Assuming this RAID layout (VE11A & C tested)
		// FileID     MeasID   	ProtName	PatName   Status    Size   	SizeOnDisk  CreationTime          CloseTime 			dependent files (fileID)
		// 11223      456   	gre			xxxxxx    cld       787776 	4194304   	14.03.2018 11:51:00   14.03.2018 11:51:03	11220
		
		for elementNumber < 9 {

			elementStr = newRaidLineSplit[i]

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
					// Scan name

					fileNameStr = elementStr
					if len(elementStr) > 2 {
						//fmt.Println(elementStr+" strcmp: %d", strings.Compare(elementStr[0:3], "Adj"))

						if elementStr[0:3] == "Adj" {
							isNotToBeCopied = 1 // borrowing retrorecon flag to not copy adjustment scans
							fmt.Println("adj")
						}
					}
				} else if elementNumber > 3 && elementNumber < 7 { // sift through possible spaces in the filename
					// Scan name (cont) 
					// Currently, xxxxxx for PatName when using anonymized raid
					// use this to identify end-of-file-name

					if elementStr != "xxxxxx" && protNameFlag == 0 {
						elementNumber -= 1
						fileNameStr = fileNameStr + "_" + elementStr
					} else if elementStr == "xxxxxx" && protNameFlag == 0 {
					  protNameFlag = 1
					}
				} else if elementNumber == 8 {
					// Date string

					date1 := elementStr
					dateStr = date1[6:10] + date1[3:5] + date1[0:2]

				} else if elementNumber == 9 {
					// Time string

					time1 := elementStr
					timeStr = reg.ReplaceAllString(time1, "")

				}

			} else {
			}
			i += 1
		}

		// stash everything into arrays (fileID, fileNameStr, copy flag ...)

		isNotToBeCopiedArray=append(isNotToBeCopiedArray, isNotToBeCopied)
		fileIDArray=append(fileIDArray,fileID)

		fileNameStr = dateStr + "_" + timeStr + "_" + "meas_" + "MID" + MeasID + "_FID" + strings.Repeat("0", 5-len(fileID)) + fileID + "_" + fileNameStr + ".dat" // get for list making purposes

		fmt.Println(fileNameStr)
		fileNameStrArray=append(fileNameStrArray, fileNameStr)


	} 	// first loop END - through raidtool dump 

	// *********************************************************************
	// SECOND LOOP - Copy oldest data first 
	// *********************************************************************

	hashlog, err := ioutil.ReadFile(*raidfilePtr)

	for j := 0; j < raidLoopCounter; j++ {
		// debug //
		// fmt.Printf("***************\n")

		// Reverse step through twix to copy oldest data first
		index := raidLoopCounter - j - 1 // number of files = raidLoopCounter

		if isNotToBeCopiedArray[index] == 0 {  // isNotToBeCopiedArray = all isNotToBeCopied's
			// // debug //
			// fmt.Printf("raidtool -m " + fileIDArray[index] + " -o " + fileNameStrArray[index] + " -a mars -p 8010 \n")
			// fmt.Printf("scp -i " + *userkeyPtr + " " + fileNameStrArray[index] +  " " + *usernamePtr + "@" + *storageaddressPtr + "\n") //change to inputs for target address and key
			// fmt.Printf("rm " + fileNameStrArray[index] + " \n")

			// *********************************************************************
			// Suitable for transfer - now check if hash exists locally >>
			// *********************************************************************

			// Download header for hashing 
			//	(hdrsignature will work on hdr, dat and dicom (and MRD))

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
				// Already copied

				fmt.Println("file ID " + fileIDArray[index] + " : Hash exists, no need to transfer") // debug //

			} else {
				// Needs to be copied

				fmt.Println("file ID " + fileIDArray[index] + " : No hash, transferring and appending to log. ***")

				// *********************************************************************
				// @ LIT wheel - Data transfer >>
				// *********************************************************************

				// download data : 
				// debug // fmt.Printf("raidtool -m " + fileID + " -o " + fileNameStr + " -a mars -p 8010 -D \n")

				cmd = exec.Command("cmd.exe", "/C", "raidtool -m "+fileIDArray[index]+" -o "+fileNameStrArray[index]+" -a mars -p 8010 -D")

				_, err = cmd.Output()
				if err != nil {
					log.Fatal(err)
				}

				// transfer data & remove host copy
				
				cmd = exec.Command("cmd.exe", "/C", "scp -i " + *userkeyPtr + " " + fileNameStrArray[index] +  " " + *usernamePtr + "@" + *storageaddressPtr)

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

				// *********************************************************************
				// append hash >>
				// *********************************************************************
				
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

		} // if @ isNotToBeCopiedArray

	} // j @ raidLoopCounter

}
