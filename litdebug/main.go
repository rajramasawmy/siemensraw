package main


// not highly efficient method to parse raidtool information
// creates raidtooltmp.txt, raidtool.txt files in the local directory
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
	usernamePtr := flag.String("user", "meduser", "username")
	storageaddressPtr := flag.String("address", "192.168.2.5:/data/LITwheel/", "storage destination address")
	debugTickPtr := flag.Int("debug", 0, "number of debug ticks")
	flag.Parse()

	// CLEAN UP USER PARAMETERS
	fmt.Println("raidfile text:", *raidfilePtr)
	fmt.Println("user key text file:", *userkeyPtr)
	fmt.Println("user:", *usernamePtr)
	fmt.Println("ticks:", *debugTickPtr)






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
// THIS IS THE START OF THE FIRST LOOP (CREATING isNotToBeCopiedArray)
// *********************************************************************

	// isNotToBeCopiedArray := []    // figure out how to initialize this stuff (arrays)
	// fileNameStrArray := []
	// fileID := []

	raidLoopCounter := 0
	for {
		fmt.Printf("**************\n")
		// debug //		fmt.Println("Reading CSV") // debug //


		if raidLoopCounter + 1 > *debugTickPtr {

			break
		} // limit how much of the RAID is processed for testing */

		if *debugTickPtr != 0 {
			raidLoopCounter += 1
		}

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

		newRaidLine := record[0]
		// debug // fmt.Println("New line read") // fmt.Println(newRaidLine + "\n++++++++++++++\n") // debug //
		if len(record[0]) < 100 { // end of file catch
			break
		}
// debug //
		// fmt.Println("New line read") //
		 // debug //
		newRaidLineSplit := strings.SplitAfterN(newRaidLine, " ", 500)

		c := "tmpstr"
		isNotToBeCopied := 0
		e := 0
		i := 0
		protNameFlag := 0
		fileID := "und" // undefined
		MeasID := "und"
		fileNameStr := "und"
		dateStr := "und"
		timeStr := "und"

		for e < 9 {



			c = newRaidLineSplit[i]
			// fmt.Printf(c + "\n")

			c = strings.Replace(c, " ", "", -1)

			if len(c) > 0 {
				// d = unicode.IsNumber(rune(c[0]))
				// fmt.Println(">>" +c)

				//		fmt.Println(IsLetter(string(c[0])))
				e += 1

				if e == 1 {
					fileID = c
				} else if e == 2 {
					if len(c) > 5 { // retrorecon jobs have 7-digit FID's, no need to download these duplicates.
						fmt.Println("retrorecon")
						isNotToBeCopied = 1
					} else {
						MeasID = strings.Repeat("0", 5-len(c)) + c
					}
				} else if e == 3 { // this should be tidied
					fileNameStr = c
					if len(c) > 2 {
						//fmt.Println(c+" strcmp: %d", strings.Compare(c[0:3], "Adj"))

						if c[0:3] == "Adj" {
							isNotToBeCopied = 1 // borrowing retrorecon flag to not copy adjustment scans
							fmt.Println("adj")
						}
					}
				} else if e > 3 && e < 7 { // sift through possible spaces in the filename

					// currently, xxxxxx for PatName when using anonymized raid
					if c != "xxxxxx" && protNameFlag == 0 {
						e -= 1
						fileNameStr = fileNameStr + "_" + c
					} else if c == "xxxxxx" && protNameFlag == 0 {
					  protNameFlag = 1
					}
				} else if e == 8 {
					//fmt.Println(c)
					date1 := c
					dateStr = date1[6:10] + date1[3:5] + date1[0:2]
					//fmt.Println("This is the date str: " + dateStr)
					/*fmt.Println(date1[6:10])
					fmt.Println(date1[3:5])
					fmt.Println(date1[0:2])
					fmt.Println(len(date1)) */
				} else if e == 9 {
					//fmt.Println(c)
					time1 := c
					timeStr = reg.ReplaceAllString(time1, "")
					//			fmt.Println("This is the time str: " + timeStr)
				}

			} else {
			}
			i += 1
		}

		// stash everything into arrays (fileID, measID, ...)
		_=isNotToBeCopied //comment this out, and put stuff here




		//	fileID := reg.ReplaceAllString(newRaidLine[0:10], "") // [0:10]-12 is affected by retrorecon, 8 is still safe with len(FILEID)=4
		fileNameStr = dateStr + "_" + timeStr + "_" + "meas_" + "MID" + MeasID + "_FID" + strings.Repeat("0", 5-len(fileID)) + fileID + "_" + fileNameStr + ".dat" // get for list making purposes
		//	fmt.Println("FILE ID: " + fileID) // debug //
		fmt.Println(fileNameStr)



	} 	// loop through raidtool dump << (raidLoop end)

	// *********************************************************************
	// THIS IS THE START OF THE SECOND LOOP (COPYING DATA BASED ON isNotToBeCopiedArray)
	// *********************************************************************





	// start loop for len(raidLoopCounter)

	// for i < raidLoopCounter {
	// 	index = raidLoopCounter - i // number of files = raidLoopCounter
	// 	if isNotToBeCopiedArray[index] == 0 {  // isNotToBeCopiedArray = all isNotToBeCopied's
	// 		fmt.Printf("raidtool -m " + fileID + " -o " + fileNameStr + " -a mars -p 8010 \n")
	fileNameStr := ""
			fmt.Printf("scp " + fileNameStr + " -i " + *userkeyPtr + " " + *usernamePtr + "@" + *storageaddressPtr + "\n") //change to inputs for target address and key
	// 		fmt.Printf("rm " + fileNameStr + " \n")
	// 		// target format: meas_MID00000_FID00000_NAME.dat
	// 	}
	// }


}
