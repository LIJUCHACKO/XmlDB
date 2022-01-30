package xmlDB

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const SEGMENTSIZE = 2000

// writeLines writes the lines to the given file.
func writeLines(DB *Database, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, item := range DB.global_dbLines {
		for _, line := range item {
			line = strings.ReplaceAll(line, "<nil:node>", "")
			line = strings.ReplaceAll(line, "</nil:node>", "")
			fmt.Fprintln(w, line)

		}
	}
	return w.Flush()
}

func readLines(path string, MaxNooflines int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var content strings.Builder

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, MaxNooflines*500) /*Assume one line on an average  contains 500 characters*/

	for scanner.Scan() {
		content.WriteString(scanner.Text())
	}
	return content.String(), scanner.Err()
}
func ReplaceHTMLSpecialEntities(input string) string {
	output := strings.Replace(input, "&amp;", "&", -1)
	output = strings.Replace(output, "&lt;", "<", -1)
	output = strings.Replace(output, "&gt;", ">", -1)
	output = strings.Replace(output, "&quot;", "\"", -1)
	output = strings.Replace(output, "&lsquo;", "‘", -1)
	output = strings.Replace(output, "&rsquo;", "’", -1)
	output = strings.Replace(output, "&tilde;", "~", -1)
	output = strings.Replace(output, "&ndash;", "–", -1)
	output = strings.Replace(output, "&mdash;", "—", -1)
	output = strings.Replace(output, "&apos;", "'", -1)

	return output
}
func ReplacewithHTMLSpecialEntities(input string) string {
	output := strings.Replace(input, "&", "&amp;", -1)
	output = strings.Replace(output, "<", "&lt;", -1)
	output = strings.Replace(output, ">", "&gt;", -1)
	output = strings.Replace(output, "\"", "&quot;", -1)
	output = strings.Replace(output, "‘", "&lsquo;", -1)
	output = strings.Replace(output, "’", "&rsquo;", -1)
	output = strings.Replace(output, "~", "&tilde;", -1)
	output = strings.Replace(output, "–", "&ndash;", -1)
	output = strings.Replace(output, "—", "&mdash;", -1)
	output = strings.Replace(output, "'", "&apos;", -1)

	return output
}

type Database struct {
	filename                 string
	removeattribute          string
	global_ids               []int
	deleted_ids              []int
	global_paths             [][]string
	global_dbLines           [][]string
	global_values            [][]string
	global_attributes        [][]string
	global_lineLastUniqueid  int
	Debug_enabled            bool
	nodeNoToLineno           []int
	pathKeylookup            [][]int
	Nodeendlookup            []int
	pathIdStack              [2000]int
	pathIdStack_index        int
	reference_linenotoinsert int
	retainid                 int
	startindex               int
	path                     string
	MaxNooflines             int
	maxInt                   int
	maxHashValue             int
	WriteLock                bool
}

func updateNodenoLineMap(DB *Database, fromLine int) {
	lineno := fromLine
	if lineno < 0 {
		lineno = 0
	}
	for {
		if lineno >= len(DB.global_ids) {
			break
		}
		id := DB.global_ids[lineno]
		if id >= 0 {
			DB.nodeNoToLineno[id] = lineno
		}

		lineno++
	}
	DB.WriteLock = false
}
func stringtono(DB *Database, line string) int {
	total := 0
	for i, ch := range line {
		total = total + (total<<5)+ int(ch)*i
	}
	if total >= DB.maxHashValue {
		total = total % DB.maxHashValue
	}
	return total
}

func suspectedLinenos(DB *Database, path string, lowerbound int, upperbound int) ([]int, []int) {

	pathParts := strings.Split(path, "/")
	var NodeNos []int
	var suspectedLineStarts []int
	var suspectedLineEnds []int
	SearchtillEnd := 0
	index := len(pathParts) - 1
	for {
		if index < 0 {
			break
		}
		part := pathParts[index]
		//fmt.Printf("\n%s keypart %s", path, part)
		if strings.Contains(part, "<") || strings.Contains(part, "..") {

			SearchtillEnd = 1

		} else {
			hashno := stringtono(DB, part)
			from := find_indexhashtable(DB, hashno, lowerbound, false)
			to := find_indexhashtable(DB, hashno, upperbound, true)
			if from >= 0 && to >= 0 {
				NodeNos = DB.pathKeylookup[hashno][from:to]
			}
			break
		}
		index--
	}
	//fmt.Printf("\nNodeNos ")
	//fmt.Println(NodeNos)
	if len(NodeNos) == 0 {
		suspectedLineStarts = append(suspectedLineStarts, 0)
		suspectedLineEnds = append(suspectedLineEnds, len(DB.global_ids))

		return suspectedLineStarts, suspectedLineEnds //1
	} else {
		for _, node := range NodeNos {
			suspectedLineStarts = append(suspectedLineStarts, DB.nodeNoToLineno[node])
			//fmt.Printf("\n DB.nodeNoToLineno[node] %d ", DB.nodeNoToLineno[node])
			if SearchtillEnd == 1 {
				suspectedLineEnds = append(suspectedLineEnds, DB.nodeNoToLineno[DB.Nodeendlookup[node]]+1)
				//fmt.Printf("\n DB.suspectedLineEnds[i] %d ", DB.suspectedLineEnds[i])
			} else {
				suspectedLineEnds = append(suspectedLineEnds, DB.nodeNoToLineno[node])
			}
		}
	}

	return suspectedLineStarts, suspectedLineEnds
}
func compare_path(current_path string, reference_path string) ([]string, []string, bool) {
	ref_pathParts := strings.Split(reference_path, "/")
	cur_pathParts := strings.Split(current_path, "/")
	len_cur_pathParts := len(cur_pathParts)
	len_ref_pathParts := len(ref_pathParts)
	cur_pathPartindex := 0
	ref_pathPartindex := 0
	skipoccured := false
	var label []string
	var value []string
	for {
		if cur_pathPartindex >= len_cur_pathParts {
			break
		}
		if ref_pathPartindex >= len_ref_pathParts {
			if skipoccured {
				return label, value, true
			}
			break
		}
		if strings.Contains(ref_pathParts[ref_pathPartindex], "<") && strings.Contains(ref_pathParts[ref_pathPartindex], ">") {
			if skipoccured {
				//no of remaining parts are matching then
				if (len(cur_pathParts) - cur_pathPartindex) > (len(ref_pathParts) - ref_pathPartindex) {
					cur_pathPartindex++
					continue
				} else {

				}
			}
			label = append(label, ref_pathParts[ref_pathPartindex])
			value = append(value, cur_pathParts[cur_pathPartindex])
			cur_pathPartindex++
			ref_pathPartindex++
			skipoccured = false
			continue
		} else if strings.Contains(ref_pathParts[ref_pathPartindex], "..") {
			skipoccured = true
			ref_pathPartindex++
			continue
		}
		if cur_pathParts[cur_pathPartindex] == ref_pathParts[ref_pathPartindex] {
			skipoccured = false
			cur_pathPartindex++
			ref_pathPartindex++
		} else {
			if skipoccured {
				cur_pathPartindex++
				continue
			} else {
				break
			}
		}

	}
	if cur_pathPartindex == len_cur_pathParts && ref_pathPartindex == len_ref_pathParts {
		return label, value, true
	} else {
		return label, value, false
	}
}
func isParentPath(parentp string, nodep string) bool {
	if len(nodep) >= len(parentp) {
		if nodep[0:len(parentp)] == parentp {
			if nodep == parentp {
				return true
			} else if len(nodep) > len(parentp) {
				if nodep[0:len(parentp)+1] == parentp+"/" {
					return true
				}
			}

		}
	}
	return false
}
func Get_common(set1 []int, set2 []int) []int {
	var result []int
	for _, element1 := range set1 {
		for _, element2 := range set2 {
			if element1 == element2 {
				result = append(result, element1)
			}
		}
	}
	return result
}
func remove(a []int, index int) []int {
	if len(a) == index { // nil or empty slice or after last element
		return a[:index]
	}
	a = append(a[:index], a[index+1:]...) // index < len(a)
	return a
}

func removeid_fromhashtable(DB *Database, hashno int, nodeId int) {
	lineno := DB.nodeNoToLineno[nodeId]
	LowLM := 0
	UpLM := len(DB.pathKeylookup[hashno]) - 1
	if UpLM < 0 {
		return
	}
	MidLM := 0
	index := -1
	for {
		MidLM = int((LowLM + UpLM) / 2)
		if lineno >= DB.nodeNoToLineno[DB.pathKeylookup[hashno][LowLM]] && lineno <= DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] {
			UpLM = MidLM
		} else if lineno > DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] && lineno <= DB.nodeNoToLineno[DB.pathKeylookup[hashno][UpLM]] {
			LowLM = MidLM
		} else {
			break
		}
		//fmt.Printf("\n%d  %d  %d", UpLM, MidLM, LowLM)
		if UpLM == LowLM || UpLM == (LowLM+1) {
			if DB.pathKeylookup[hashno][UpLM] == nodeId {
				index = UpLM
				break
			}
			if DB.pathKeylookup[hashno][LowLM] == nodeId {
				index = LowLM
				break
			}
			break
		}

	}
	if index < 0 {
		return
	}
	//fmt.Printf("\nremoveid_fromhashtable-%d %d", DB.pathKeylookup[hashno][index], index)
	DB.pathKeylookup[hashno] = append(DB.pathKeylookup[hashno][:index], DB.pathKeylookup[hashno][index+1:]...) // index < len(a)
	//fmt.Println(DB.pathKeylookup[hashno])
	return
}
func find_indexhashtable(DB *Database, hashno int, node_lineno int, roof bool) int {
	LowLM := 0
	UpLM := len(DB.pathKeylookup[hashno]) - 1
	if UpLM < 0 {
		return -1
	}
	MidLM := 0
	index := -1
	//node_lineno := DB.nodeNoToLineno[nodeId]

	for {
		MidLM = int((LowLM + UpLM) / 2)
		if roof {
			if node_lineno >= DB.nodeNoToLineno[DB.pathKeylookup[hashno][LowLM]] && node_lineno < DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] {
				if DB.nodeNoToLineno[DB.pathKeylookup[hashno][LowLM]] == DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] {
					LowLM = MidLM
				} else {
					UpLM = MidLM
				}

			} else if node_lineno >= DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] && node_lineno <= DB.nodeNoToLineno[DB.pathKeylookup[hashno][UpLM]] {
				if DB.nodeNoToLineno[DB.pathKeylookup[hashno][UpLM]] == DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] {
					UpLM = MidLM
				} else {
					LowLM = MidLM
				}

			} else {
				break
			}
		} else {
			if node_lineno >= DB.nodeNoToLineno[DB.pathKeylookup[hashno][LowLM]] && node_lineno <= DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] {
				if DB.nodeNoToLineno[DB.pathKeylookup[hashno][LowLM]] == DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] {
					LowLM = MidLM
				} else {
					UpLM = MidLM
				}

			} else if node_lineno > DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] && node_lineno <= DB.nodeNoToLineno[DB.pathKeylookup[hashno][UpLM]] {
				if DB.nodeNoToLineno[DB.pathKeylookup[hashno][UpLM]] == DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] {
					UpLM = MidLM
				} else {
					LowLM = MidLM
				}

			} else {
				break
			}
		}

		//fmt.Printf("\n%d  %d  %d", UpLM, MidLM, LowLM)
		if UpLM == LowLM || UpLM == (LowLM+1) {
			index = UpLM
			break
		}

	}
	if index < 0 {
		if node_lineno < DB.nodeNoToLineno[DB.pathKeylookup[hashno][0]] {
			index = 0
		} else {
			index = len(DB.pathKeylookup[hashno])
		}

	}
	return index
}
func insertid_intohashtable(DB *Database, hashno int, nodeId int) {

	//fmt.Printf("\ninsertid_intohashtable-entry")
	if len(DB.pathKeylookup[hashno]) == 0 {
		DB.pathKeylookup[hashno] = append(DB.pathKeylookup[hashno], nodeId)
		return
	}
	//fmt.Println(DB.pathKeylookup[hashno])
	lineno := DB.reference_linenotoinsert
	//fmt.Println(nodeId)
	LowLM := 0
	UpLM := len(DB.pathKeylookup[hashno]) - 1
	MidLM := 0
	index := -1
	if DB.nodeNoToLineno[DB.pathKeylookup[hashno][LowLM]] == DB.nodeNoToLineno[DB.pathKeylookup[hashno][UpLM]] {
		if DB.nodeNoToLineno[DB.pathKeylookup[hashno][LowLM]] == lineno {
			DB.pathKeylookup[hashno] = append(DB.pathKeylookup[hashno], nodeId)
			return
		}
	}
	for {
		MidLM = int((LowLM + UpLM) / 2)
		if lineno >= DB.nodeNoToLineno[DB.pathKeylookup[hashno][LowLM]] && lineno < DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] {
			if DB.nodeNoToLineno[DB.pathKeylookup[hashno][LowLM]] == DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] {
				LowLM = MidLM
			} else {
				UpLM = MidLM
			}

		} else if lineno >= DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] && lineno <= DB.nodeNoToLineno[DB.pathKeylookup[hashno][UpLM]] {
			if DB.nodeNoToLineno[DB.pathKeylookup[hashno][UpLM]] == DB.nodeNoToLineno[DB.pathKeylookup[hashno][MidLM]] {
				UpLM = MidLM
			} else {
				LowLM = MidLM
			}

		} else {
			break
		}
		//fmt.Printf("\n%d  %d  %d", UpLM, MidLM, LowLM)
		if UpLM == LowLM || UpLM == (LowLM+1) {
			index = UpLM
			break
		}

	}
	if index < 0 {
		if lineno < DB.nodeNoToLineno[DB.pathKeylookup[hashno][0]] {
			index = 0
		} else {
			DB.pathKeylookup[hashno] = append(DB.pathKeylookup[hashno], nodeId)
			//fmt.Printf("\ninsertid_intohashtable")
			//fmt.Println(DB.pathKeylookup[hashno])
			return
		}

	}
	DB.pathKeylookup[hashno] = append(DB.pathKeylookup[hashno][:index+1], DB.pathKeylookup[hashno][index:]...) // index < len(a)
	DB.pathKeylookup[hashno][index] = nodeId
	//fmt.Printf("\ninsertid_intohashtable-%d", index)
	//fmt.Println(DB.pathKeylookup[hashno])
	return
}
func remove_string(a []string, index int) []string {
	if len(a) == index { // nil or empty slice or after last element
		return a[:index]
	}
	a = append(a[:index], a[index+1:]...) // index < len(a)
	return a
}
func insert(a []int, index int, value int) []int {
	if len(a) == index { // nil or empty slice or after last element
		return append(a, value)
	}
	a = append(a[:index+1], a[index:]...) // index < len(a)
	a[index] = value
	return a
}
func insert_string(a []string, index int, value string) []string {
	if len(a) == index { // nil or empty slice or after last element
		return append(a, value)
	}
	a = append(a[:index+1], a[index:]...) // index < len(a)
	a[index] = value
	return a
}

func getSegmenNoIndex(DB *Database, index int) (int, int) {
	size := 0
	for seg, item := range DB.global_dbLines {
		if size+len(item) > index {
			return seg, index - size
		}
		size = size + len(item)
	}
	if size >= index {
		return len(DB.global_dbLines) - 1, len(DB.global_dbLines[len(DB.global_dbLines)-1])
	}
	return -1, -1
}
func formatxml(lines []string) []string {
	newlines := []string{}
	level := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line[0:2] == "</" {

		} else if line[0:1] == "<" && strings.Contains(line, "</") {
			level++
		} else if line[0:1] == "<" {
			level++
		}

		space := ""
		i := 1
		for {
			if i >= level {
				break
			}
			space = space + "  "
			i++
		}
		line_n := space + line
		newlines = append(newlines, line_n)
		if line[0:2] == "</" {
			level--
		} else if line[0:1] == "<" && strings.Contains(line, "</") {
			level--
		} else if line[len(line)-2:] == "/>" {
			level--
		} else if line[0:2] == "<!" {
			level--
		}
	}
	return newlines
}
func fill_DBdata(DB *Database, dbline string, value string, attribute string, NodeName string, mode int) int {

	update_path(DB, NodeName, mode)
	if NodeName[0] == '!' && DB.global_lineLastUniqueid == 0 {
		DB.global_lineLastUniqueid = -1
	}
	unique_id := DB.global_lineLastUniqueid
	if DB.startindex < 0 {
		//fmt.Printf("\n%s %s %d %s %s", DB.path, dbline, unique_id, value, attribute)
		if len(DB.global_dbLines) == 0 {
			DB.global_dbLines = append(DB.global_dbLines, []string{})
			DB.global_values = append(DB.global_values, []string{})
			DB.global_attributes = append(DB.global_attributes, []string{})
			DB.global_paths = append(DB.global_paths, []string{})
		}
		lastsegment := len(DB.global_dbLines) - 1
		if len(DB.global_dbLines[lastsegment]) >= SEGMENTSIZE {
			DB.global_dbLines = append(DB.global_dbLines, []string{})
			DB.global_values = append(DB.global_values, []string{})
			DB.global_attributes = append(DB.global_attributes, []string{})
			DB.global_paths = append(DB.global_paths, []string{})
			lastsegment++
		}

		DB.global_dbLines[lastsegment] = append(DB.global_dbLines[lastsegment], dbline)
		DB.global_values[lastsegment] = append(DB.global_values[lastsegment], value)
		DB.global_attributes[lastsegment] = append(DB.global_attributes[lastsegment], attribute)
		DB.global_paths[lastsegment] = append(DB.global_paths[lastsegment], DB.path)

		DB.global_ids = append(DB.global_ids, unique_id)

		DB.global_lineLastUniqueid++
		if DB.global_lineLastUniqueid >= DB.maxInt {
			fmt.Printf("load_db: Total no. of Uniqueid>= DB.MaxNooflines, Please increase DB.MaxNooflines before loading db")
			os.Exit(1)
		}
	} else {

		if DB.retainid >= 0 {
			unique_id = DB.retainid
			DB.retainid = -1
		} else {
			if DB.global_lineLastUniqueid >= DB.maxInt {
				if len(DB.deleted_ids) > 0 {
					unique_id = DB.deleted_ids[0]
					DB.deleted_ids = DB.deleted_ids[1:]
				} else {
					fmt.Printf("InsertAtLine: Total no. of Uniqueid>= DB.maxInt, Please increase DB.maxInt")
					os.Exit(1)
				}
			}
		}
		SegNo, index := getSegmenNoIndex(DB, DB.startindex)
		DB.global_dbLines[SegNo] = insert_string(DB.global_dbLines[SegNo], index, dbline)
		DB.global_values[SegNo] = insert_string(DB.global_values[SegNo], index, value)
		DB.global_attributes[SegNo] = insert_string(DB.global_attributes[SegNo], index, attribute)
		DB.global_paths[SegNo] = insert_string(DB.global_paths[SegNo], index, DB.path)

		DB.global_ids = insert(DB.global_ids, DB.startindex, unique_id)

		/*splitting segments*/
		if len(DB.global_dbLines[SegNo]) >= SEGMENTSIZE*2 {

			if SegNo < len(DB.global_dbLines)-1 {
				DB.global_dbLines = append(DB.global_dbLines[:SegNo+2], DB.global_dbLines[SegNo+1:]...)          // index < len(a)
				DB.global_values = append(DB.global_values[:SegNo+2], DB.global_values[SegNo+1:]...)             // index < len(a)
				DB.global_attributes = append(DB.global_attributes[:SegNo+2], DB.global_attributes[SegNo+1:]...) // index < len(a)
				DB.global_paths = append(DB.global_paths[:SegNo+2], DB.global_paths[SegNo+1:]...)                // index < len(a)                     // index < len(a)
			} else {
				DB.global_dbLines = append(DB.global_dbLines, []string{})
				DB.global_values = append(DB.global_values, []string{})
				DB.global_attributes = append(DB.global_attributes, []string{})
				DB.global_paths = append(DB.global_paths, []string{})
			}
			DB.global_dbLines[SegNo+1] = DB.global_dbLines[SegNo][SEGMENTSIZE:]
			DB.global_values[SegNo+1] = DB.global_values[SegNo][SEGMENTSIZE:]
			DB.global_attributes[SegNo+1] = DB.global_attributes[SegNo][SEGMENTSIZE:]
			DB.global_paths[SegNo+1] = DB.global_paths[SegNo][SEGMENTSIZE:]

			DB.global_dbLines[SegNo] = DB.global_dbLines[SegNo][:SEGMENTSIZE]
			DB.global_values[SegNo] = DB.global_values[SegNo][:SEGMENTSIZE]
			DB.global_attributes[SegNo] = DB.global_attributes[SegNo][:SEGMENTSIZE]
			DB.global_paths[SegNo] = DB.global_paths[SegNo][:SEGMENTSIZE]

		}
		if DB.Debug_enabled {
			fmt.Printf("insertatline :Inserting New Node %d\n", unique_id)
		}
		DB.startindex++
		if DB.global_lineLastUniqueid < DB.maxInt && unique_id == DB.global_lineLastUniqueid {
			DB.global_lineLastUniqueid++
			if DB.global_lineLastUniqueid >= DB.maxInt {
				fmt.Printf("load_db: Total no. of Uniqueid>= DB.MaxNooflines, Please increase DB.MaxNooflines before loading db")
				os.Exit(1)
			}
		}
	}
	if unique_id >= 0 {
		if mode < 3 {
			Node_hash := stringtono(DB, NodeName)
			insertid_intohashtable(DB, Node_hash, unique_id)
		}
		if mode == 1 {
			DB.pathIdStack[DB.pathIdStack_index] = unique_id
			DB.pathIdStack_index++
		} else if mode == 2 {
			DB.Nodeendlookup[unique_id] = unique_id
		} else if mode == 3 {
			DB.pathIdStack_index--
			DB.Nodeendlookup[DB.pathIdStack[DB.pathIdStack_index]] = unique_id

		}
	}
	//fmt.Printf("\n%s %d %d %d", DB.path, DB.pathIdStack_index, unique_id, mode)
	return unique_id
}
func update_path(DB *Database, NodeName string, mode int) {
	//fmt.Printf(" path- %s  line-%s mode-%d\n", path, NodeName, mode)
	//1.Add 2.Add and Remove 3.Remove
	if len(DB.path) > 3 {
		if DB.path[len(DB.path)-2:len(DB.path)] == "/~" {
			DB.path = DB.path[0 : len(DB.path)-2]
		}
	}

	if len(DB.removeattribute) > 0 {
		if DB.path[len(DB.path)-len(DB.removeattribute):] == DB.removeattribute {
			DB.path = DB.path[0 : len(DB.path)-len(DB.removeattribute)]
			DB.removeattribute = ""
		}
	}
	if mode >= 2 {
		DB.removeattribute = "/" + NodeName
	}
	if mode <= 2 {
		DB.path = DB.path + "/" + NodeName
	}

	if mode == 3 {
		//path = path[0 : len(path)-len(NodeName)-1]
		DB.path = DB.path + "/~"
	}
	//fmt.Printf(" path- %s  line-%s \n", DB.path, line)
}
func parseAndLoadXml(DB *Database, content string) []int {
	DB.WriteLock = true
	nodes := []int{}
	var nodeStart strings.Builder
	var attributebuffer strings.Builder
	var valuebuffer strings.Builder
	nodeEnded := false
	CommentStarted := false
	Comment2Started := false
	xmldeclarationStarted := false
	CDATAStarted := false
	NodeName := ""
	lastindex := 0
	index := 0
	for {
		if content[index] == '<' {
			if !CommentStarted && !CDATAStarted && !xmldeclarationStarted {
				if content[index+1] != '/' {
					if nodeStart.Len() > 0 {
						//////
						node := fill_DBdata(DB, nodeStart.String(), valuebuffer.String(), strings.ReplaceAll(attributebuffer.String(), "\"\"", "\""), NodeName, 1)
						if DB.startindex >= 0 {
							nodes = append(nodes, node)
						}
						valuebuffer.Reset()
						attributebuffer.Reset()
						/////
					}
					nodeStart.Reset()
					nodeStart.WriteString("<nil:node>")
					nodeStart.WriteString(content[lastindex:index])
					nodeStart.WriteString("</nil:node>")
					if len(strings.TrimSpace(content[lastindex:index])) > 0 {
						////
						node := fill_DBdata(DB, nodeStart.String(), content[lastindex:index], "", "nil:node", 2)
						if DB.startindex >= 0 {
							nodes = append(nodes, node)
						}
						////
					}
					nodeStart.Reset()
					lastindex = index
					nodeEnded = false
				}
				if content[index+1] == '!' {
					//comparestringForward(line, "<![CDATA[")
					nodeEnded = false
					if content[index+2] == '[' {
						lastindex = index
						nodeStart.Reset()
						CDATAStarted = true
						lastindex = index
					} else if content[index+2] == '-' {
						CommentStarted = true
						lastindex = index
					} else {
						Comment2Started = true
						lastindex = index
					}
				} else if content[index+1] == '?' {
					//comparestringForward(line, "<?")
					nodeEnded = false
					xmldeclarationStarted = true
					lastindex = index
				} else if content[index+1] == '/' {
					//comparestringForward(line, "</")
					nodeEnded = true
					if nodeStart.Len() > 0 {
						nodeStart.WriteString(content[lastindex:index])
						valuebuffer.WriteString(content[lastindex:index])
						lastindex = index
					} else {
						nodeStart.WriteString("<nil:node>")
						nodeStart.WriteString(content[lastindex:index])
						nodeStart.WriteString("</nil:node>")
						if len(strings.TrimSpace(content[lastindex:index])) > 0 {
							////
							node := fill_DBdata(DB, nodeStart.String(), content[lastindex:index], "", "nil:node", 2)
							if DB.startindex >= 0 {
								nodes = append(nodes, node)
							}
							////
						}
						nodeStart.Reset()
						lastindex = index
					}
				} else {

				}
			}
		}
		if content[index] == '>' {
			if CommentStarted {
				if content[index-1] == '-' {
					//comparestringBackward(line, "->")
					CommentStarted = false
					buffer := content[lastindex : index+1]
					lastindex = index + 1
					if len(strings.TrimSpace(buffer)) > 0 {
						////
						node := fill_DBdata(DB, buffer, valuebuffer.String(), strings.ReplaceAll(attributebuffer.String(), "\"\"", "\""), "!COMMENT!", 2)
						if DB.startindex >= 0 {
							nodes = append(nodes, node)
						}
						valuebuffer.Reset()
						attributebuffer.Reset()
						////
					}
				}
			} else if CDATAStarted {
				//comparestringBackward(line, "]]>")
				if content[index-1] == ']' {
					buffer := content[lastindex : index+1]
					lastindex = index + 1
					if len(strings.TrimSpace(buffer)) > 0 {
						////
						node := fill_DBdata(DB, buffer, valuebuffer.String(), strings.ReplaceAll(attributebuffer.String(), "\"\"", "\""), "!CDATA!", 2)
						if DB.startindex >= 0 {
							nodes = append(nodes, node)
						}
						valuebuffer.Reset()
						attributebuffer.Reset()
						////
					}
					CDATAStarted = false
				}
			} else if xmldeclarationStarted {
				//comparestringBackward(line, "?>"
				if content[index-1] == '?' {
					xmldeclarationStarted = false
					buffer := content[lastindex : index+1]
					lastindex = index + 1
					if len(strings.TrimSpace(buffer)) > 0 {
						////
						node := fill_DBdata(DB, buffer, valuebuffer.String(), strings.ReplaceAll(attributebuffer.String(), "\"\"", "\""), "!XMLDECL!", 2)
						if DB.startindex >= 0 {
							nodes = append(nodes, node)
						}
						valuebuffer.Reset()
						attributebuffer.Reset()
						////
					}
				}
			} else if Comment2Started {
				Comment2Started = false
				buffer := content[lastindex : index+1]
				lastindex = index + 1
				if len(strings.TrimSpace(buffer)) > 0 {
					///
					node := fill_DBdata(DB, buffer, valuebuffer.String(), strings.ReplaceAll(attributebuffer.String(), "\"\"", "\""), "!COMMENT2!", 2)
					if DB.startindex >= 0 {
						nodes = append(nodes, node)
					}
					valuebuffer.Reset()
					attributebuffer.Reset()
					////
				}
			} else {

				//extract attribute
				if content[index-1] == '/' {
					//if comparestringBackward(line, "/>", index) {
					nodeStart.WriteString(content[lastindex : index+1])
					parts := strings.Split(strings.TrimSpace(content[lastindex+1:index-1]), "\" ")
					for partind, part := range parts {
						if partind > 0 {
							if len(strings.TrimSpace(part)) > 0 {
								if attributebuffer.Len() > 1 {
									attributebuffer.WriteString("||")
								}
								attributebuffer.WriteString(strings.TrimSpace(part) + "\"")
							}
						} else {
							subparts := strings.Split(part, " ")
							if len(subparts) > 1 {
								attributebuffer.WriteString(strings.TrimSpace(subparts[1]) + "\"")
							}
							NodeName = subparts[0]
						}
					}
					lastindex = index + 1
					///
					node := fill_DBdata(DB, nodeStart.String(), valuebuffer.String(), strings.ReplaceAll(attributebuffer.String(), "\"\"", "\""), NodeName, 2)
					if DB.startindex >= 0 {
						nodes = append(nodes, node)
					}
					valuebuffer.Reset()
					attributebuffer.Reset()
					///
					nodeStart.Reset()
					nodeEnded = false

				} else {
					//} else if comparestringBackward(line, ">", index) {
					if nodeEnded {
						///
						if nodeStart.Len() > 0 {
							nodeStart.WriteString(content[lastindex : index+1])
							node := fill_DBdata(DB, nodeStart.String(), valuebuffer.String(), strings.ReplaceAll(attributebuffer.String(), "\"\"", "\""), strings.TrimSpace(content[lastindex+2:index]), 2)
							if DB.startindex >= 0 {
								nodes = append(nodes, node)
							}
						} else {
							node := fill_DBdata(DB, content[lastindex:index+1], valuebuffer.String(), strings.ReplaceAll(attributebuffer.String(), "\"\"", "\""), strings.TrimSpace(content[lastindex+2:index]), 3)
							if DB.startindex >= 0 {
								nodes = append(nodes, node)
							}
						}

						////

						valuebuffer.Reset()
						attributebuffer.Reset()
						nodeStart.Reset()
						nodeEnded = false
					} else {
						nodeStart.WriteString(content[lastindex : index+1])
						parts := strings.Split(strings.TrimSpace(content[lastindex+1:index]), "\" ")
						for partind, part := range parts {
							if partind > 0 {
								if len(strings.TrimSpace(part)) > 0 {
									if attributebuffer.Len() > 1 {
										attributebuffer.WriteString("||")
									}
									attributebuffer.WriteString(strings.TrimSpace(part) + "\"")
								}
							} else {
								subparts := strings.Split(part, " ")
								if len(subparts) > 1 {
									attributebuffer.WriteString(strings.TrimSpace(subparts[1]) + "\"")
								}
								NodeName = subparts[0]
							}
						}
					}
					lastindex = index + 1
				}
			}
		}
		index++
		if index >= len(content) {
			break
		}
	}

	if len(strings.TrimSpace(content[lastindex:index])) > 0 {
		fmt.Printf("xml is corrupt")
		os.Exit(1)
	}
	if CommentStarted || xmldeclarationStarted || Comment2Started || CDATAStarted {
		fmt.Printf("xml is corrupt")
		os.Exit(1)
	}
	if len(strings.TrimSpace(nodeStart.String())) > 0 {
		fmt.Printf("xml is corrupt")
		os.Exit(1)
	}
	//fmt.Printf("\n %d  %d %d  %d", len(newlines), len(values), len(attributes), len(paths))
	return nodes
}

func NodeLine(DB *Database, nodeId int) int {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-NodeLine\n")
	}
	lineno := DB.nodeNoToLineno[nodeId]
	if lineno < 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		lineno = -1
	}
	return lineno
}
func NodeEnd(DB *Database, nodeId int) int {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-NodeEnd\n")
	}
	lineno := DB.nodeNoToLineno[nodeId]
	if lineno < 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		return -1
	}

	if DB.Nodeendlookup[nodeId] >= 0 {

		lineno = DB.nodeNoToLineno[DB.Nodeendlookup[nodeId]] + 1

	} else {
		fmt.Printf("Warning :node  doesnot exist\n")
		lineno = -1
	}
	return lineno

}
func Dump_DB(DB *Database) string {
	var final strings.Builder
	for _, item := range DB.global_dbLines {
		for _, line := range item {
			line = strings.ReplaceAll(line, "<nil:node>", "")
			line = strings.ReplaceAll(line, "</nil:node>", "")
			final.WriteString(line)

		}
	}
	return final.String()
}
func SaveAs_DB(DB *Database, filename string) {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-Save_DB\n")
	}
	//lines := formatxml(DB.global_dbLines)
	err := writeLines(DB, filename)
	if err != nil {
		fmt.Printf("Cannot save db  : %s\n", err)

	}
}
func Save_DB(DB *Database) {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-Save_DB\n")
	}
	if len(DB.filename) == 0 {
		fmt.Printf("Filename not specified\n")
		return
	}
	//lines := formatxml(DB.global_dbLines)
	err := writeLines(DB, DB.filename)
	if err != nil {
		fmt.Printf("Cannot save db  : %s\n", err)

	}
}
func Load_dbcontent(DB *Database, xmllines []string) {
	/* xmllines should not contain /n /r*/
	var contentByte strings.Builder
	for _, line := range xmllines {
		contentByte.WriteString(line)
	}
	text := contentByte.String()
	if len(text) < 2000 {
		if !validatexml(text) {
			fmt.Println(text)
			fmt.Println("Load_dbcontent-XML not valid ,DB not loaded")
			return
		}
	}

	load_xmlstring(DB, text)
}
func load_xmlstring(DB *Database, content string) {
	if DB.MaxNooflines < 99999 {
		DB.MaxNooflines = 99999
	}
	DB.maxInt = DB.MaxNooflines
	DB.nodeNoToLineno = make([]int, DB.maxInt)
	DB.maxHashValue=97343
	DB.pathKeylookup = make([][]int, DB.maxHashValue)
	DB.Nodeendlookup = make([]int, DB.maxInt)
	DB.startindex = -1
	DB.retainid = -1
	DB.pathIdStack_index = 0
	DB.global_ids = make([]int, 0, DB.maxInt)
	DB.global_paths = make([][]string, 0, 10)
	DB.global_attributes = make([][]string, 0, 10)
	DB.global_values = make([][]string, 0, 10)
	DB.global_dbLines = make([][]string, 0, 10)
	DB.global_lineLastUniqueid = 0
	DB.removeattribute = ""
	DB.reference_linenotoinsert = 0

	DB.path = ""
	parseAndLoadXml(DB, content)

	if DB.Debug_enabled {
		fmt.Printf("load_db : over\n")
	}

	updateNodenoLineMap(DB, 0)
	//fmt.Println(DB.global_ids)
	if DB.Debug_enabled {
		fmt.Printf("load_db :xml db loaded\n No of nodes-%d\n", DB.global_lineLastUniqueid)
		//var final strings.Builder
		i := 0
		for seg, item := range DB.global_dbLines {
			for index, line := range item {
				nodeend := 0
				nodebeg := 0
				if DB.global_ids[i] >= 0 {
					nodebeg = DB.nodeNoToLineno[DB.global_ids[i]]
					nodeend = DB.nodeNoToLineno[DB.Nodeendlookup[DB.global_ids[i]]] + 1
				}
				fmt.Printf("\n path- %s  line- %s  nodeid-%d nodebeg-%d nodeend-%d", DB.global_paths[seg][index], line, DB.global_ids[i], nodebeg, nodeend)
				fmt.Printf("\n value- %s attribute-%s", DB.global_values[seg][index], DB.global_attributes[seg][index])
				i++
			}
		}

	}

}
func Load_db(DB *Database, filename string) error {
	DB.filename = filename
	lines, err := readLines(filename, DB.MaxNooflines)
	if err != nil {
		fmt.Printf("Cannot load_db :Read : %s\n", err)
		return err
	}
	load_xmlstring(DB, lines)
	return nil
}
func GetAllNodeAttributes(DB *Database, nodeId int) ([]string, []string) {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-GetNodeAttribute\n")
	}
	var labels []string
	var values []string
	LineNo := DB.nodeNoToLineno[nodeId]
	if LineNo < 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		return labels, values
	}

	SegNo, index := getSegmenNoIndex(DB, LineNo)
	attributes := strings.Split(DB.global_attributes[SegNo][index], "||")
	for _, attri := range attributes {
		attri := strings.TrimSpace(attri)
		LabelValue := strings.Split(attri, "=\"")
		if len(LabelValue) >= 2 {
			labels = append(labels, LabelValue[0])
			values = append(values, LabelValue[1][:len(LabelValue[1])-1])
		}
	}
	return labels, values
}
func GetNodeAttribute(DB *Database, nodeId int, label string) string {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-GetNodeAttribute\n")
	}
	LineNo := DB.nodeNoToLineno[nodeId]
	if LineNo < 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		return ""
	}
	SegNo, index := getSegmenNoIndex(DB, LineNo)
	attributes := strings.Split(DB.global_attributes[SegNo][index], "||")
	for _, attri := range attributes {
		attri := strings.TrimSpace(attri)
		LabelValue := strings.Split(attri, "=\"")
		if len(LabelValue) >= 2 {
			if LabelValue[0] == strings.TrimSpace(label) {
				Value := LabelValue[1]
				//removing end quotes
				//fmt.Printf("Value %s\n", Value)
				return Value[:len(Value)-1]
			}
		}
	}
	return ""
}
func GetNodeValue(DB *Database, nodeId int) string {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-GetNodeValue\n")
	}
	lineno := DB.nodeNoToLineno[nodeId]
	if lineno < 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		return ""
	}
	SegNo, index := getSegmenNoIndex(DB, lineno)
	return DB.global_values[SegNo][index]
}
func GetNodeName(DB *Database, nodeId int) string {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-GetNodeName\n")
	}
	lineno := DB.nodeNoToLineno[nodeId]
	if lineno < 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		return ""
	}
	SegNo, index := getSegmenNoIndex(DB, lineno)
	path := DB.global_paths[SegNo][index]
	pathparts := strings.Split(path, "/")
	return pathparts[len(pathparts)-1]
}
func UpdateNodevalue(DB *Database, nodeId int, new_value string) ([]int, error) {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-UpdateNodevalue\n")
	}

	nodes, err := update_nodevalue(DB, nodeId, ReplacewithHTMLSpecialEntities(new_value))

	return nodes, err
}

func update_nodevalue(DB *Database, nodeId int, new_value string) ([]int, error) {
	Nooflines := (NodeEnd(DB, nodeId) - NodeLine(DB, nodeId))
	if Nooflines > 2 {
		fmt.Printf("Error :Cannot update value- Node contains subnodes\n")
		return []int{}, errors.New("Cannot update value- Node contains subnodes")
	}
	content := GetNodeContents(DB, nodeId)
	content = strings.ReplaceAll(content, "\n", "-")
	content = strings.ReplaceAll(content, "><", ">-<")
	if len(content) == 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		return []int{}, errors.New("node  doesnot exist")
	}
	value := GetNodeValue(DB, nodeId)
	result := ""
	if len(value) == 0 && Nooflines == 1 {
		if strings.Contains(content, "/>") {
			parts := strings.Split(content, "/>")
			result = parts[0] + ">" + new_value + "</" + GetNodeName(DB, nodeId) + ">"
		}
	} else {
		parts := strings.Split(content, ">")
		if len(parts) > 1 {
			part1 := parts[1]
			part1parts := strings.Split(part1, "<")
			result = parts[0] + ">" + new_value + "<" + part1parts[1] + ">"
		}
	}
	//fmt.Printf("\n new content %s\n", result)
	replacednodes, err := replaceNodeRetainid(DB, nodeId, result)
	if DB.Debug_enabled {
		fmt.Printf("UpdateNodevalue :Updating node %d\n", nodeId)
		fmt.Printf("%s\n", GetNodeContents(DB, nodeId))
	}

	return replacednodes, err
}
func UpdateAttributevalue(DB *Database, nodeId int, label string, value string) ([]int, error) {
	value = strings.TrimSpace(value)
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-UpdateAttributevalue\n")
	}
	beginning := NodeLine(DB, nodeId)
	if beginning < 0 {
		fmt.Printf("Warning :node  doesnot exist\n")

		return []int{}, errors.New("node  doesnot exist")
	}
	SegNo, index := getSegmenNoIndex(DB, beginning)
	content := DB.global_dbLines[SegNo][index]
	NodeWithoutValue := strings.Contains(content, "/>")
	contentparts := []string{}
	if NodeWithoutValue {
		contentparts = strings.Split(content, "/>")
	} else {
		contentparts = strings.Split(content, ">")
	}

	contentparts0 := contentparts[0]
	if strings.Contains(contentparts[0], label+"=") {
		oldvalue := GetNodeAttribute(DB, nodeId, label)
		if DB.Debug_enabled {
			fmt.Printf("replacing -%s -by- %s", label+"=\""+oldvalue+"\"", label+"=\""+value+"\"")
		}
		if len(value) > 0 {
			contentparts0 = strings.ReplaceAll(contentparts0, label+"=\""+oldvalue+"\"", label+"=\""+value+"\"")
		} else {
			contentparts0 = strings.ReplaceAll(contentparts0, label+"=\""+oldvalue+"\"", "")
		}

	} else {
		if len(value) > 0 {
			contentparts0 = (contentparts0 + " " + label + "=\"" + value + "\"")
		}
	}
	contentnew := ""
	if NodeWithoutValue {
		contentnew = contentparts0 + "/>"
	} else {
		contentnew = contentparts0 + ">"
	}

	for i, part := range contentparts {
		if i > 0 && len(strings.TrimSpace(part)) > 0 {
			contentnew = contentnew + part + ">"
		}
	}
	DB.global_dbLines[SegNo][index] = contentnew
	parts := strings.Split(strings.TrimSpace(contentparts0), "\" ")
	var attributebuffer strings.Builder
	for partind, part := range parts {
		if partind > 0 {
			if len(strings.TrimSpace(part)) > 0 {
				if attributebuffer.Len() > 1 {
					attributebuffer.WriteString("||")
				}
				attributebuffer.WriteString(strings.TrimSpace(part) + "\"")
			}
		} else {
			subparts := strings.Split(part, " ")
			if len(subparts) > 1 {
				attributebuffer.WriteString(strings.TrimSpace(subparts[1]) + "\"")
			}
		}

	}
	DB.global_attributes[SegNo][index] = strings.ReplaceAll(attributebuffer.String(), "\"\"", "\"")
	if DB.Debug_enabled {
		fmt.Printf("UpdateNodevalue :Updating node %d\n", nodeId)
		fmt.Printf("%s\n", GetNodeContents(DB, nodeId))
	}
	replacednodes := []int{}
	replacednodes = append(replacednodes, nodeId)
	return replacednodes, nil
}
func GetNodeContentRaw(DB *Database, nodeId int) string {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-GetNodeContents\n")
	}

	var Output strings.Builder
	beginning := NodeLine(DB, nodeId)
	if beginning < 0 {
		return Output.String()
	}
	end := NodeEnd(DB, nodeId)
	if DB.Debug_enabled {
		fmt.Printf("getNodeContentsRaw :Fetching Contents from line %d to %d \n", beginning, end)
	}
	i := beginning
	for i < end {
		SegNo, index := getSegmenNoIndex(DB, i)
		//lines = append(lines, DB.global_dbLines[SegNo][index])
		Output.WriteString(DB.global_dbLines[SegNo][index])
		Output.WriteString("\n")
		i++
	}

	return Output.String()
}
func GetNodeContents(DB *Database, nodeId int) string {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-GetNodeContents\n")
	}

	var Output strings.Builder
	beginning := NodeLine(DB, nodeId)
	if beginning < 0 {
		return Output.String()
	}
	end := NodeEnd(DB, nodeId)
	if DB.Debug_enabled {
		fmt.Printf("getNodeContents :Fetching Contents from line %d to %d \n", beginning, end)
	}
	var lines []string
	i := beginning
	for i < end && i < beginning+200 {
		SegNo, index := getSegmenNoIndex(DB, i)
		lines = append(lines, DB.global_dbLines[SegNo][index])
		i++
	}

	lines = formatxml(lines)

	for _, line := range lines {
		line = strings.ReplaceAll(line, "<nil:node>", "")
		line = strings.ReplaceAll(line, "</nil:node>", "")
		Output.WriteString(line)
		Output.WriteString("\n")
	}
	if (end - beginning) > 200 {
		Output.WriteString("\n .....Remaining lines are not printed......\n ")
	}
	return Output.String()
}
func RemoveNode(DB *Database, nodeId int) []int {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-RemoveNode\n")
	}

	nodes := remove_Node(DB, nodeId)

	updateNodenoLineMap(DB, DB.startindex)
	return nodes
}

func remove_Lines(DB *Database, start int, end int) {
	//            *        *
	//   ---------=========--------
	DB.global_ids = append(DB.global_ids[:start], DB.global_ids[end:]...)
	startSegNo, startIndex := getSegmenNoIndex(DB, start)
	endSegNo, endIndex := getSegmenNoIndex(DB, end)
	if startSegNo == endSegNo {
		//with in segment
		DB.global_dbLines[startSegNo] = append(DB.global_dbLines[startSegNo][:startIndex], DB.global_dbLines[startSegNo][endIndex:]...)
		DB.global_paths[startSegNo] = append(DB.global_paths[startSegNo][:startIndex], DB.global_paths[startSegNo][endIndex:]...)
		DB.global_values[startSegNo] = append(DB.global_values[startSegNo][:startIndex], DB.global_values[startSegNo][endIndex:]...)
		DB.global_attributes[startSegNo] = append(DB.global_attributes[startSegNo][:startIndex], DB.global_attributes[startSegNo][endIndex:]...)
	} else {
		DB.global_dbLines[startSegNo] = DB.global_dbLines[startSegNo][:startIndex]
		DB.global_paths[startSegNo] = DB.global_paths[startSegNo][:startIndex]
		DB.global_values[startSegNo] = DB.global_values[startSegNo][:startIndex]
		DB.global_attributes[startSegNo] = DB.global_attributes[startSegNo][:startIndex]

		DB.global_dbLines[endSegNo] = DB.global_dbLines[endSegNo][endIndex:]
		DB.global_paths[endSegNo] = DB.global_paths[endSegNo][endIndex:]
		DB.global_values[endSegNo] = DB.global_values[endSegNo][endIndex:]
		DB.global_attributes[endSegNo] = DB.global_attributes[endSegNo][endIndex:]

		//remove in between segments
		if endSegNo > (startSegNo + 1) {
			DB.global_dbLines = append(DB.global_dbLines[:startSegNo+1], DB.global_dbLines[endSegNo:]...)
			DB.global_paths = append(DB.global_paths[:startSegNo+1], DB.global_paths[endSegNo:]...)
			DB.global_values = append(DB.global_values[:startSegNo+1], DB.global_values[endSegNo:]...)
			DB.global_attributes = append(DB.global_attributes[:startSegNo+1], DB.global_attributes[endSegNo:]...)
		}

	}

}

func remove_Node(DB *Database, nodeId int) []int {

	if DB.Debug_enabled {
		fmt.Printf("removeNode :Removing node %d\n", nodeId)
	}

	startindex := NodeLine(DB, nodeId)
	end := NodeEnd(DB, nodeId)
	var removedids []int
	DB.startindex = startindex
	DB.WriteLock = true
	for ind := startindex; ind < end; ind++ {
		SegNo, index := getSegmenNoIndex(DB, ind)
		path := DB.global_paths[SegNo][index]
		path_parts := strings.Split(path, "/")
		if path_parts[len(path_parts)-1] != "~" {
			hashno := stringtono(DB, path_parts[len(path_parts)-1])
			removeid_fromhashtable(DB, hashno, DB.global_ids[ind])
		}

		DB.deleted_ids = append(DB.deleted_ids, DB.global_ids[ind])
		removedids = append(removedids, DB.global_ids[ind])
		DB.nodeNoToLineno[DB.global_ids[ind]] = -1

	}
	remove_Lines(DB, startindex, end)
	return removedids
}

func validatexml(content string) bool {
	nodesnames := []string{}
	nodeEnded := false
	CommentStarted := false
	Comment2Started := false
	xmldeclarationStarted := false
	CDATAStarted := false
	lastindex := 0
	index := 0
	for {
		if content[index] == '<' {
			if !CommentStarted && !CDATAStarted && !xmldeclarationStarted {
				if content[index+1] == '!' {
					nodeEnded = false
					if content[index+2] == '[' {
						lastindex = index + 3
						CDATAStarted = true
						lastindex = index
					} else if content[index+2] == '-' {
						CommentStarted = true
						lastindex = index + 3
					} else {
						Comment2Started = true
						lastindex = index + 2
					}
				} else if content[index+1] == '?' {
					nodeEnded = false
					xmldeclarationStarted = true
					lastindex = index + 2
				} else if content[index+1] == '/' {
					nodeEnded = true
					lastindex = index + 2
				} else {
					lastindex = index + 1
				}
			}
		}
		if content[index] == '>' {
			if CommentStarted {
				if content[index-1] == '-' {
					CommentStarted = false
					lastindex = index + 1
				}
			} else if CDATAStarted {
				if content[index-1] == ']' {
					lastindex = index + 1
					CDATAStarted = false
				}
			} else if xmldeclarationStarted {
				if content[index-1] == '?' {
					xmldeclarationStarted = false
					lastindex = index + 1
				}
			} else if Comment2Started {
				Comment2Started = false
				lastindex = index + 1
			} else {
				if content[index-1] == '/' {
					lastindex = index + 1
					nodeEnded = false
				} else {
					if nodeEnded {
						if len(nodesnames) == 0 {
							return false
						}
						if strings.TrimSpace(nodesnames[len(nodesnames)-1]) != strings.TrimSpace(content[lastindex:index]) {
							return false
						}
						nodesnames = nodesnames[0 : len(nodesnames)-1]
						nodeEnded = false
					} else {
						parts := strings.Split(strings.TrimSpace(content[lastindex:index]), " ")
						nodesnames = append(nodesnames, parts[0])
					}
					lastindex = index + 1
				}
			}
		}
		index++
		if index >= len(content) {
			break
		}
	}
	if len(strings.TrimSpace(content[lastindex:index])) > 0 {
		fmt.Println(content[lastindex:index])
		return false
	}
	if CommentStarted || xmldeclarationStarted || Comment2Started || CDATAStarted {
		fmt.Println("Comment/xmldeclaration/CDATA session not closed")
		return false
	}
	if len(nodesnames) > 0 {
		fmt.Println("Nodes not closed")
		fmt.Println(nodesnames)
		return false
	}
	return true
}
func insertAtLine(DB *Database, lineno int, sub_xml string, retainid int) ([]int, error) {
	DB.retainid = retainid
	DB.removeattribute = ""
	DB.pathIdStack_index = 0
	DB.startindex = lineno
	startindex_tmp := lineno
	if lineno > 0 {
		DB.reference_linenotoinsert = lineno - 1
		SegNo, index := getSegmenNoIndex(DB, lineno-1)
		path := DB.global_paths[SegNo][index]
		if path[len(path)-2:len(path)] == "/~" {
			path = path[0 : len(path)-2]
		}
		if strings.Contains(DB.global_dbLines[SegNo][index], "</") || strings.Contains(DB.global_dbLines[SegNo][index], "/>") || strings.Contains(DB.global_dbLines[SegNo][index], "<!") {
			path_parts := strings.Split(path, "/")
			path = path[0 : len(path)-len(path_parts[len(path_parts)-1])-1]
		}
		DB.path = path
	} else {
		DB.reference_linenotoinsert = 0
		DB.path = ""
	}
	newlines := strings.Split(sub_xml, "\n")
	var contentByte strings.Builder
	for _, line := range newlines {
		contentByte.WriteString(line)
	}
	content := contentByte.String()
	if !validatexml(content) {
		fmt.Printf("\n xml content is not proper- aborting insertion")
		return []int{}, errors.New("xml content is not proper- aborting insertion")
	}
	nodes := parseAndLoadXml(DB, content)

	updateNodenoLineMap(DB, startindex_tmp-1)
	DB.startindex = -1
	return nodes, nil
}
func ReplaceNode(DB *Database, nodeId int, sub_xml string) ([]int, error) {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-ReplaceNode\n")
	}

	if DB.Debug_enabled {
		fmt.Printf("replaceNode :Replacing node %d\n", nodeId)
	}
	if !validatexml(sub_xml) {
		fmt.Printf("\n xml content is not proper- aborting replacing")

		return []int{}, errors.New("xml content is not proper- aborting replacing")
	}
	startindex := NodeLine(DB, nodeId)
	if startindex < 0 {
		return []int{}, errors.New("Node doesnot exists")
	}
	rmids := remove_Node(DB, nodeId)
	if len(rmids) > 0 {
		nodes, err := insertAtLine(DB, startindex, sub_xml, -1)

		return nodes, err
	}
	return []int{}, errors.New("Node not found")
}
func replaceNodeRetainid(DB *Database, nodeId int, sub_xml string) ([]int, error) {
	if DB.Debug_enabled {
		fmt.Printf("replaceNodeRetainid :Replacing node %d\n", nodeId)
	}
	if !validatexml(sub_xml) {
		fmt.Printf("\n xml content is not proper- aborting replacing")

		return []int{}, errors.New("xml content is not proper- aborting replacing")
	}
	startindex := NodeLine(DB, nodeId)
	if startindex < 0 {
		return []int{}, errors.New("Node doesnot exists")
	}
	removed := remove_Node(DB, nodeId)
	DB.deleted_ids = remove(DB.deleted_ids, len(DB.deleted_ids)-len(removed))
	nodes, err := insertAtLine(DB, startindex, sub_xml, removed[0])
	return nodes, err
}
func IslowestNode(DB *Database, nodeId int) bool {
	end := NodeEnd(DB, nodeId)
	if (end - NodeLine(DB, nodeId)) == 1 {
		return true
	}
	return false
}
func CutPasteAsSubNode(DB *Database, UnderId int, nodeId int) error {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-CutPasteAsSubNode\n")
	}
	previousparentid := ParentNode(DB, nodeId)
	if previousparentid == -1 {
		fmt.Println("Node doesnot exists")
		return errors.New("Node doesnot exists")
	}
	//fmt.Printf("\nnodeid %d parentid %d DB.nodeNoToLineno[nodeId] %d\n", nodeId, previousparentid, DB.nodeNoToLineno[nodeId])
	pSegNo, pindex := getSegmenNoIndex(DB, NodeLine(DB, previousparentid))
	previousparentpath := DB.global_paths[pSegNo][pindex]
	//remove from old location
	Line := NodeLine(DB, nodeId)
	end := NodeEnd(DB, nodeId)
	startindex := Line

	DB_global_ids := []int{}
	DB_global_dbLines := []string{}
	DB_global_paths := []string{}
	DB_global_values := []string{}
	DB_global_attributes := []string{}
	SegNo, index := getSegmenNoIndex(DB, startindex)
	DB.WriteLock = true
	for Line < end {
		DB_global_ids = append(DB_global_ids, DB.global_ids[startindex])
		DB.global_ids = remove(DB.global_ids, startindex)

		DB_global_dbLines = append(DB_global_dbLines, DB.global_dbLines[SegNo][index])
		DB.global_dbLines[SegNo] = remove_string(DB.global_dbLines[SegNo], index)
		DB_global_paths = append(DB_global_paths, DB.global_paths[SegNo][index])
		DB.global_paths[SegNo] = remove_string(DB.global_paths[SegNo], index)
		DB_global_values = append(DB_global_values, DB.global_values[SegNo][index])
		DB.global_values[SegNo] = remove_string(DB.global_values[SegNo], index)
		DB_global_attributes = append(DB_global_attributes, DB.global_attributes[SegNo][index])
		DB.global_attributes[SegNo] = remove_string(DB.global_attributes[SegNo], index)
		Line++

	}
	updateNodenoLineMap(DB, 0)

	ToNodeend := NodeEnd(DB, UnderId)
	if ToNodeend == -1 {
		fmt.Println("New Parent Node doesnot exists")
		return errors.New("New Parent Node doesnot exists")
	}
	NewParentNodename := GetNodeName(DB, UnderId)
	NewParentNodeisEmpty := false
	insertLine := ToNodeend - 1
	SegNop, indexp := getSegmenNoIndex(DB, insertLine)
	if (ToNodeend - NodeLine(DB, UnderId)) == 1 {
		if strings.Contains(DB.global_dbLines[SegNop][indexp], "/>") {
			NewParentNodeisEmpty = true
			DB.global_dbLines[SegNop][indexp] = strings.ReplaceAll(DB.global_dbLines[SegNop][indexp], "/>", ">")
		} else {
			fmt.Println(" Node is a lowest node , not a nil node")
			return errors.New(" Node is a lowest node , not a nil node")
		}

	}
	//prepare initial path
	if NewParentNodeisEmpty {
		SegNop, indexp = getSegmenNoIndex(DB, insertLine)
	} else {
		SegNop, indexp = getSegmenNoIndex(DB, insertLine-1)
	}

	newparentpath := DB.global_paths[SegNop][indexp]
	if newparentpath[len(newparentpath)-2:len(newparentpath)] == "/~" {
		newparentpath = newparentpath[0 : len(newparentpath)-2]
	}
	if strings.Contains(DB.global_dbLines[SegNop][indexp], "</") || strings.Contains(DB.global_dbLines[SegNop][indexp], "/>") || strings.Contains(DB.global_dbLines[SegNop][indexp], "<!") {
		path_parts := strings.Split(newparentpath, "/")
		newparentpath = newparentpath[0 : len(newparentpath)-len(path_parts[len(path_parts)-1])-1]
	}
	//Paste to new location
	Line = 0
	if NewParentNodeisEmpty {
		insertLine++
	}
	DB.WriteLock = true
	for Line < len(DB_global_dbLines) {
		//SegNo, index := getSegmenNoIndex(DB, Line)
		DB.path = newparentpath + strings.ReplaceAll(DB_global_paths[Line], previousparentpath, "")

		newSegNo, newindex := getSegmenNoIndex(DB, insertLine)
		DB.global_dbLines[newSegNo] = insert_string(DB.global_dbLines[newSegNo], newindex, DB_global_dbLines[Line])
		DB.global_values[newSegNo] = insert_string(DB.global_values[newSegNo], newindex, DB_global_values[Line])
		DB.global_attributes[newSegNo] = insert_string(DB.global_attributes[newSegNo], newindex, DB_global_attributes[Line])
		DB.global_paths[newSegNo] = insert_string(DB.global_paths[newSegNo], newindex, DB.path)
		DB.global_ids = insert(DB.global_ids, insertLine, DB_global_ids[Line])
		insertLine++

		Line++
	}
	if NewParentNodeisEmpty {

		newSegNo, newindex := getSegmenNoIndex(DB, insertLine)
		DB.global_dbLines[newSegNo] = insert_string(DB.global_dbLines[newSegNo], newindex, "</"+NewParentNodename+">")
		DB.global_values[newSegNo] = insert_string(DB.global_values[newSegNo], newindex, "")
		DB.global_attributes[newSegNo] = insert_string(DB.global_attributes[newSegNo], newindex, "")
		DB.global_paths[newSegNo] = insert_string(DB.global_paths[newSegNo], newindex, newparentpath+"/~")

		DB.global_ids = insert(DB.global_ids, insertLine, DB.global_lineLastUniqueid)
		DB.Nodeendlookup[UnderId] = DB.global_lineLastUniqueid
		DB.global_lineLastUniqueid++
		if DB.global_lineLastUniqueid >= DB.maxInt {
			fmt.Printf("load_db: Total no. of Uniqueid>= DB.MaxNooflines, Please increase DB.MaxNooflines before loading db")
			os.Exit(1)
		}
	}
	updateNodenoLineMap(DB, 0)

	DB.startindex = -1
	return nil
}
func InserSubNode(DB *Database, nodeId int, sub_xml string) ([]int, error) {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-InserSubNode\n")
	}

	if !validatexml(sub_xml) {
		fmt.Printf("\n xml content is not proper- aborting InserSubNode")

		return []int{}, errors.New("xml content is not proper- aborting InserSubNode")
	}

	end := NodeEnd(DB, nodeId)
	if (end - NodeLine(DB, nodeId)) == 1 {
		nodes, err := update_nodevalue(DB, nodeId, sub_xml)
		nodes = nodes[1:]
		return nodes, err
	}
	if end < 0 {
		fmt.Printf("Error :node  doesnot exist\n")

		return []int{}, errors.New("node  doesnot exist")
	}
	nodes, err := insertAtLine(DB, end-1, sub_xml, -1)

	return nodes, err
}
func AppendAfterNode(DB *Database, nodeId int, sub_xml string) ([]int, error) {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-AppendAfterNode\n")
	}

	if !validatexml(sub_xml) {
		fmt.Printf("\nError : xml content is not proper- aborting AppendAfterNode")

		return []int{}, errors.New("xml content is not proper- aborting ")
	}
	end := NodeEnd(DB, nodeId)
	if end < 0 {
		fmt.Printf("Error :node  doesnot exist\n")

		return []int{}, errors.New("node  doesnot exist")
	}
	nodes, err := insertAtLine(DB, end, sub_xml, -1)

	return nodes, err
}
func AppendBeforeNode(DB *Database, nodeId int, sub_xml string) ([]int, error) {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-AppendBeforeNode\n")
	}

	if !validatexml(sub_xml) {
		fmt.Printf("\n xml content is not proper- aborting AppendBeforeNode")

		return []int{}, errors.New("xml content is not proper- aborting ")
	}
	start := NodeLine(DB, nodeId)
	if start < 0 {
		fmt.Printf("Error :node  doesnot exist\n")

		return []int{}, errors.New("node  doesnot exist")
	}
	nodes, err := insertAtLine(DB, start, sub_xml, -1)

	return nodes, err
}
func LocateRequireParentdNode(DB *Database, parent_nodeLine int, RequiredPath string, LineNo_inp int) int {
	//Search 'required node' backward
	if LineNo_inp < 0 || parent_nodeLine < 0 {
		return -1
	}
	SegNo, index := getSegmenNoIndex(DB, parent_nodeLine)
	ParentPath := DB.global_paths[SegNo][index]

	suspectedLineStarts, _ := suspectedLinenos(DB, RequiredPath, parent_nodeLine, LineNo_inp+1)
	if DB.Debug_enabled {
		fmt.Printf("#####LocateRequireParentdNode###\n ")
		fmt.Printf("ParentPath- %s\n", ParentPath)
		fmt.Printf("LineNo %d\n", LineNo_inp)
		fmt.Printf("RequiredPath %s\n", RequiredPath)
		fmt.Printf("parent_nodeLine %d\n", parent_nodeLine)
	}

	//locate line just above LineNo_inp
	requiredline := 0
	//for _, start := range Starts {
	i := 0
	for _, start := range suspectedLineStarts {

		if start >= parent_nodeLine && start <= LineNo_inp {
			if start > requiredline {
				requiredline = start
			}
		}
		i++
	}
	SegNo, index = getSegmenNoIndex(DB, requiredline)
	if len(DB.global_paths[SegNo][index]) >= len(ParentPath) {

		_, _, stat := compare_path(DB.global_paths[SegNo][index], RequiredPath)

		if stat {
			if DB.Debug_enabled {
				fmt.Printf("Located parent %d\n", DB.global_ids[requiredline])
				fmt.Printf("====LocateRequireParentdNode===\n ")
			}
			return DB.global_ids[requiredline]
		} else {
			return -1

		}
	} else {
		return -1

	}

	return -1
}

func locateNodeLine(DB *Database, parent_nodeLine int, QUERY string, RegExp string, onlypath bool, isRegExp bool) ([]int, []string) {

	parent_endline := DB.nodeNoToLineno[DB.Nodeendlookup[DB.global_ids[parent_nodeLine]]] + 1

	var ResultIds []int
	var Label_Values []string
	if parent_nodeLine < 0 {
		return ResultIds, Label_Values
	}
	InsideParent := true
	SegNo, index := getSegmenNoIndex(DB, parent_nodeLine)
	ParentPath := DB.global_paths[SegNo][index]
	if len(QUERY) > 0 {
		QUERY = ParentPath + "/" + QUERY
	} else {
		QUERY = ParentPath
	}

	if DB.Debug_enabled {
		fmt.Printf("####Locate Node#####\n")
		fmt.Printf("QUERY- %s\n", QUERY)
	}

	QueryPath := strings.ReplaceAll(QUERY, "*", "")
	if DB.Debug_enabled {
		fmt.Printf("ParentPath- %s\n", ParentPath)
		fmt.Printf("QueryPATH- %s\n", QueryPath)
		fmt.Printf("Search Value %s\n", RegExp)

	}

	suspectedLineStarts, suspectedLineEnds := suspectedLinenos(DB, QueryPath, parent_nodeLine, parent_endline)
	for index, start := range suspectedLineStarts {
		if start >= parent_nodeLine && start <= parent_endline {
			LineNo := start

			for InsideParent && LineNo < len(DB.global_ids) && LineNo <= suspectedLineEnds[index] {
				//fmt.Printf("\nDB.global_paths[LineNo] %s ParentPath %s\n", DB.global_paths[LineNo], ParentPath)
				SegNo, index := getSegmenNoIndex(DB, LineNo)
				if isParentPath(ParentPath, DB.global_paths[SegNo][index]) {

					labels, values, path_matching := compare_path(DB.global_paths[SegNo][index], QueryPath)

					labelValueStr := ""

					for index2, label := range labels {
						labelValueStr = labelValueStr + label + "=" + values[index2] + ";"
					}
					if path_matching {
						//fmt.Printf("\npath matching %s", QueryPath)
						if onlypath {
							ResultIds = append(ResultIds, LineNo)
							if DB.Debug_enabled {
								fmt.Printf(" Query matching -lineno %d\n", LineNo)
							}
							Label_Values = append(Label_Values, labelValueStr)
						} else {
							//iterate through all
							values_attributes := strings.Split(RegExp, ";")
							all_satisfied := true
							for _, valueorAttribute := range values_attributes {
								valueorAttribute = strings.TrimSpace(valueorAttribute)
								if len(valueorAttribute) > 0 {

									if strings.Contains(RegExp, "=\"") {

										if len(strings.TrimSpace(DB.global_attributes[SegNo][index])) == 0 {
											all_satisfied = false
										} else {
											attributes := strings.Split(DB.global_attributes[SegNo][index], "||")
											attrib_matching := false
											for _, attrib := range attributes {
												attrib = strings.TrimSpace(attrib)
												if len(attrib) > 0 {
													match := false
													if isRegExp {
														match, _ = regexp.MatchString(valueorAttribute, attrib)
													} else {
														match = (valueorAttribute == attrib)

													}
													if match {
														attrib_matching = true
													}

												} else {

												}
											}
											if !attrib_matching {

												all_satisfied = false
											}
										}

									} else {

										match := false
										if isRegExp {
											match, _ = regexp.MatchString(valueorAttribute, DB.global_values[SegNo][index])
										} else {
											valueorAttribute = ReplacewithHTMLSpecialEntities(valueorAttribute)
											match = (valueorAttribute == strings.TrimSpace(DB.global_values[SegNo][index]))
										}
										if !match {
											all_satisfied = false
										}

									}
								}
							}
							if all_satisfied {
								ResultIds = append(ResultIds, LineNo)
								Label_Values = append(Label_Values, labelValueStr)
								if DB.Debug_enabled {
									fmt.Printf("Query  matching -lineno %d\n", LineNo)
								}
							}

						}

					}
				} else {
					InsideParent = false
				}
				LineNo++
			}
		}

	}
	if DB.Debug_enabled {
		fmt.Printf("===LocateNode===\n")
	}
	return ResultIds, Label_Values
}
func ParentNode(DB *Database, nodeId int) int {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-ParentNode\n")
	}
	LineNo := DB.nodeNoToLineno[nodeId]
	ResultId := -1
	if nodeId < 0 {
		return ResultId
	}
	if LineNo < 0 {
		return ResultId
	}
	SegNo, index := getSegmenNoIndex(DB, LineNo)
	NodePath := DB.global_paths[SegNo][index]
	parts := strings.Split(NodePath, "/")

	RequiredPath := NodePath[0 : len(NodePath)-len(parts[len(parts)-1])-1]
	return LocateRequireParentdNode(DB, NodeLine(DB, 0), RequiredPath, LineNo)
}

func ChildNodes(DB *Database, nodeId int) []int {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-ChildNodes\n")
	}
	LineNo := DB.nodeNoToLineno[nodeId]
	var ResultIds []int
	if nodeId < 0 {
		return ResultIds
	}
	if LineNo < 0 {
		return ResultIds
	}
	SegNo, index := getSegmenNoIndex(DB, LineNo)
	NodePath := DB.global_paths[SegNo][index]
	nodeDepth := len(strings.Split(NodePath, "/"))
	Node_end := DB.nodeNoToLineno[DB.Nodeendlookup[nodeId]] + 1
	InsideParent := true
	LineNo++
	for InsideParent && LineNo < Node_end {
		SegNo, index := getSegmenNoIndex(DB, LineNo)
		//fmt.Printf("\npath-%s ", DB.global_paths[LineNo])
		if isParentPath(NodePath, DB.global_paths[SegNo][index]) {
			if DB.global_paths[SegNo][index][len(DB.global_paths[SegNo][index])-2:] == "/~" {

			} else {
				if len(strings.Split(DB.global_paths[SegNo][index], "/")) == nodeDepth+1 {
					ResultIds = append(ResultIds, DB.global_ids[LineNo])
				}

			}
		} else {
			InsideParent = false

		}
		LineNo++
	}
	return ResultIds
}

func NextNode(DB *Database, nodeId int) int {
	LineNo := DB.nodeNoToLineno[nodeId]
	if LineNo < 0 {
		return -1
	}
	SegNo, index := getSegmenNoIndex(DB, LineNo)
	NodePath := DB.global_paths[SegNo][index]
	nodeDepth := len(strings.Split(NodePath, "/"))
	Node_end := DB.nodeNoToLineno[DB.Nodeendlookup[nodeId]] + 1
	nextnodeid := DB.global_ids[Node_end]
	SegNo, index = getSegmenNoIndex(DB, Node_end)
	nextNodePath := DB.global_paths[SegNo][index]
	if DB.path[len(nextNodePath)-2:len(nextNodePath)] == "/~" {
		return -1
	}
	nextnodeDepth := len(strings.Split(nextNodePath, "/"))
	if nodeDepth == nextnodeDepth {
		return nextnodeid
	} else {
		return -1
	}

}

func separateValue(pathpart string) (string, string) {
	i := strings.Index(pathpart, "[")
	path := pathpart
	value := ""
	if i > -1 {
		path = pathpart[0:i]
		value = pathpart[i+1 : len(pathpart)-1]
	}

	return path, value
}

func preparePathparts(path string) []string {
	result := []string{}
	i := strings.Index(path, "]/")
	for i > -1 {
		result = append(result, path[0:i+1])
		path = path[i+2:]
		i = strings.Index(path, "]/")
	}
	result = append(result, path)
	return result
}

func pathwithoutvalue(path string) string {
	output := ""
	items := preparePathparts(path)
	for _, item := range items {
		pathpart, _ := separateValue(item)
		if len(output) == 0 {
			output = pathpart
		} else {
			output = output + "/" + pathpart
		}

	}
	return output
}

func GetNode(DB *Database, parent_nodeId int, QUERY_inp string) ([]int, []string) {
	// ldld/dkdicmk/<xe>/kjk[]/lkl/
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-GetNode\n")
	}
	if DB.Debug_enabled {

		fmt.Printf("==Process Query===\n")
		fmt.Printf("ProcessQuery :QUERY_inp- %s\n", QUERY_inp)
	}
	RequiredPath := pathwithoutvalue(QUERY_inp)
	if strings.Contains(RequiredPath, "*") {
		QUERY_Parts := strings.Split(RequiredPath, "*")
		RequiredPath = QUERY_Parts[0]
	}

	if DB.Debug_enabled {
		fmt.Printf("ProcessQuery :QUERY- %s\n", QUERY_inp)
		fmt.Printf("ProcessQuery :RequiredPath- %s\n", RequiredPath)

	}
	var labels_result []string
	var final_nodesLineNo []int
	parent_nodeLine := NodeLine(DB, parent_nodeId)
	if parent_nodeLine < 0 {
		return []int{}, []string{}
	}
	final_nodesLineNo = append(final_nodesLineNo, parent_nodeLine)
	labels_result = append(labels_result, "")
	parts := preparePathparts(QUERY_inp)
	for _, part := range parts {
		QUERYSTR, RegExp := separateValue(part)
		QUERYSTR = strings.ReplaceAll(QUERYSTR, "*", "")
		if len(part) > 0 {
			var nextnodesLineNo []int
			var nextlabels []string

			for ind, node := range final_nodesLineNo {
				onlypath := true
				if len(RegExp) > 0 {
					onlypath = false
				}
				if QUERYSTR[0:1] == "/" {
					QUERYSTR = QUERYSTR[1:]
				}

				isRegExp := false
				//RegExp starts with > eg :- [>[a-b]{1,2}]
				if len(RegExp) > 0 {
					if RegExp[0:1] == ">" {
						RegExp = RegExp[1:]
						isRegExp = true
					}
				}
				identifiedNodes, labels := locateNodeLine(DB, node, QUERYSTR, RegExp, onlypath, isRegExp)

				for i, label := range labels {

					nextlabels = append(nextlabels, label+labels_result[ind])
					nextnodesLineNo = append(nextnodesLineNo, identifiedNodes[i])
					if DB.Debug_enabled {
						fmt.Printf("ProcessQuery :label %s\n", label)
						fmt.Printf("ProcessQuery :identifiedNode %d\n", identifiedNodes[i])

					}

				}

			}
			labels_result = nextlabels
			final_nodesLineNo = nextnodesLineNo
		}
	}
	var ResultIds []int
	for index, label_res := range labels_result {
		nodeLine := final_nodesLineNo[index]
		if nodeLine >= 0 {
			RequiredPathN := RequiredPath
			//fmt.Printf("ProcessQuery :label_res %s\n", label_res)
			entries := strings.Split(label_res, ";")
			for _, entry := range entries {
				parts := strings.Split(entry, "=")
				if len(parts[0]) > 0 {
					RequiredPathN = strings.ReplaceAll(RequiredPathN, parts[0], parts[1])
					//fmt.Printf("ProcessQuery : %s  %s\n", parts[0], parts[1])
				}
			}
			//fmt.Printf("ProcessQuery :parent_nodeLine %s\n", DB.global_paths[parent_nodeLine]+"/"+RequiredPathN)
			SegNo, index := getSegmenNoIndex(DB, parent_nodeLine)
			ResultId := LocateRequireParentdNode(DB, parent_nodeLine, DB.global_paths[SegNo][index]+"/"+RequiredPathN, nodeLine)
			if ResultId > 0 {
				if DB.Debug_enabled {
					fmt.Printf("ProcessQuery :ResultId %d\n", ResultId)
				}
				ResultIds = append(ResultIds, ResultId)
			}
		}

	}
	return ResultIds, labels_result
}
func NodeDebug(DB *Database, nodeId int) {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-GetNodeContents\n")
	}

	i := NodeLine(DB, nodeId)
	if i < 0 {
		return
	}
	end := NodeEnd(DB, nodeId)
	if DB.Debug_enabled {
		fmt.Printf("getNodeContents :Fetching Contents from line %d to %d \n", i, end)
	}
	for {
		if i >= end {
			break
		}
		SegNo, index := getSegmenNoIndex(DB, i)
		line := DB.global_dbLines[SegNo][index]
		nodeend := 0
		nodebeg := 0
		if DB.global_ids[i] >= 0 {
			nodebeg = DB.nodeNoToLineno[DB.global_ids[i]]
			nodeend = DB.nodeNoToLineno[DB.Nodeendlookup[DB.global_ids[i]]] + 1
		}

		fmt.Printf("\n path- %s  line- %s  nodeid-%d nodebeg-%d nodeend-%d", DB.global_paths[SegNo][index], line, DB.global_ids[i], nodebeg, nodeend)

		fmt.Printf("\n value- %s attribute-%s", DB.global_values[SegNo][index], DB.global_attributes[SegNo][index])
		i++

	}

	return
}
