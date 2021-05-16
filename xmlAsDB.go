package xmlDB

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

//const maxInt = 9999999

// writeLines writes the lines to the given file.
func writeLines(DB *Database, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range DB.global_dbLines {
		line = strings.ReplaceAll(line, "<nil:node>", "")
		line = strings.ReplaceAll(line, "</nil:node>", "")
		fmt.Fprintln(w, line+"\r")
	}
	return w.Flush()
}

func readLines(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var content strings.Builder

	scanner := bufio.NewScanner(file)
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
	global_paths             []string
	global_dbLines           []string
	global_values            []string
	global_attributes        []string
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
	WriteLock                bool
}

func updateNodenoLineMap(DB *Database, fromLine int) {
	lineno := fromLine
	for {
		if lineno >= len(DB.global_dbLines) {
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
		total = total + int(ch)*i
	}
	if total > DB.maxInt {
		total = total - DB.maxInt
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
		if index <= 0 {
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
			NodeNos = DB.pathKeylookup[hashno][from:to]
			break
		}
		index--
	}
	//fmt.Printf("\nNodeNos ")
	//fmt.Println(NodeNos)
	if len(NodeNos) == 0 {
		suspectedLineStarts = append(suspectedLineStarts, 0)
		suspectedLineEnds = append(suspectedLineEnds, len(DB.global_dbLines))

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
		DB.pathKeylookup[hashno] = append(DB.pathKeylookup[hashno], nodeId)
		return
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
		DB.global_dbLines = append(DB.global_dbLines, dbline)
		DB.global_values = append(DB.global_values, value)
		DB.global_attributes = append(DB.global_attributes, attribute)
		DB.global_paths = append(DB.global_paths, DB.path)
		DB.global_ids = append(DB.global_ids, unique_id)

		DB.global_lineLastUniqueid++
		if DB.global_lineLastUniqueid >= DB.maxInt {
			fmt.Printf("load_db: Total no. of Uniqueid>= DB.MaxNooflines, Please increase DB.MaxNooflines before loading db")
			os.Exit(1)
		}
	} else {

		if DB.retainid > 0 {
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
		DB.global_dbLines = insert_string(DB.global_dbLines, DB.startindex, dbline)
		DB.global_values = insert_string(DB.global_values, DB.startindex, value)
		DB.global_attributes = insert_string(DB.global_attributes, DB.startindex, attribute)
		DB.global_ids = insert(DB.global_ids, DB.startindex, unique_id)
		DB.global_paths = insert_string(DB.global_paths, DB.startindex, DB.path)
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
						node := fill_DBdata(DB, nodeStart.String(), valuebuffer.String(), attributebuffer.String(), NodeName, 1)
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
						node := fill_DBdata(DB, nodeStart.String(), strings.TrimSpace(content[lastindex:index]), "", "nil:node", 2)
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
						valuebuffer.WriteString(strings.TrimSpace(content[lastindex:index]))
						lastindex = index
					} else {
						nodeStart.WriteString("<nil:node>")
						nodeStart.WriteString(content[lastindex:index])
						nodeStart.WriteString("</nil:node>")
						if len(strings.TrimSpace(content[lastindex:index])) > 0 {
							////
							node := fill_DBdata(DB, nodeStart.String(), strings.TrimSpace(content[lastindex:index]), "", "nil:node", 2)
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
						node := fill_DBdata(DB, buffer, valuebuffer.String(), attributebuffer.String(), "!COMMENT!", 2)
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
						node := fill_DBdata(DB, buffer, valuebuffer.String(), attributebuffer.String(), "!CDATA!", 2)
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
						node := fill_DBdata(DB, buffer, valuebuffer.String(), attributebuffer.String(), "!XMLDECL!", 2)
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
					node := fill_DBdata(DB, buffer, valuebuffer.String(), attributebuffer.String(), "!COMMENT2!", 2)
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
					parts := strings.Split(strings.TrimSpace(content[lastindex+1:index-1]), " ")
					for partind, part := range parts {
						if partind > 0 {
							if len(strings.TrimSpace(part)) > 0 {
								if attributebuffer.Len() > 1 {
									attributebuffer.WriteString("||")
								}
								attributebuffer.WriteString(strings.TrimSpace(part))
							}
						} else {
						}
					}
					lastindex = index + 1
					///
					node := fill_DBdata(DB, nodeStart.String(), valuebuffer.String(), attributebuffer.String(), parts[0], 2)
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
							node := fill_DBdata(DB, nodeStart.String(), valuebuffer.String(), attributebuffer.String(), strings.TrimSpace(content[lastindex+2:index]), 2)
							if DB.startindex >= 0 {
								nodes = append(nodes, node)
							}
						} else {
							node := fill_DBdata(DB, content[lastindex:index+1], valuebuffer.String(), attributebuffer.String(), strings.TrimSpace(content[lastindex+2:index]), 3)
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
						parts := strings.Split(strings.TrimSpace(content[lastindex+1:index]), " ")
						for partind, part := range parts {
							if partind > 0 {
								if len(strings.TrimSpace(part)) > 0 {
									if attributebuffer.Len() > 1 {
										attributebuffer.WriteString("||")
									}
									attributebuffer.WriteString(strings.TrimSpace(part))
								}
							} else {
								NodeName = part
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
	var contentByte strings.Builder
	for _, line := range xmllines {
		contentByte.WriteString(line)
	}
	load_xmlstring(DB, contentByte.String())
}
func load_xmlstring(DB *Database, content string) {
	if DB.MaxNooflines < 99999 {
		DB.MaxNooflines = 99999
	}
	DB.maxInt = DB.MaxNooflines
	DB.nodeNoToLineno = make([]int, DB.maxInt)
	DB.pathKeylookup = make([][]int, DB.maxInt)
	DB.Nodeendlookup = make([]int, DB.maxInt)
	DB.startindex = -1
	DB.retainid = -1
	DB.pathIdStack_index = 0
	DB.global_ids = make([]int, 0, DB.maxInt)
	DB.global_paths = make([]string, 0, DB.maxInt)
	DB.global_attributes = make([]string, 0, DB.maxInt)
	DB.global_values = make([]string, 0, DB.maxInt)
	DB.global_dbLines = make([]string, 0, DB.maxInt)
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

		for i, line := range DB.global_dbLines {
			nodeend := 0
			nodebeg := 0
			if DB.global_ids[i] >= 0 {
				nodebeg = DB.nodeNoToLineno[DB.global_ids[i]]
				nodeend = DB.nodeNoToLineno[DB.Nodeendlookup[DB.global_ids[i]]] + 1
			}

			fmt.Printf("\n path- %s  line- %s  nodeid-%d nodebeg-%d nodeend-%d", DB.global_paths[i], line, DB.global_ids[i], nodebeg, nodeend)
			fmt.Printf("\n value- %s attribute-%s", DB.global_values[i], DB.global_attributes[i])
		}
	}

}
func Load_db(DB *Database, filename string) error {
	DB.filename = filename
	lines, err := readLines(filename)
	if err != nil {
		fmt.Printf("Cannot load_db :Read : %s\n", err)
		return err
	}
	load_xmlstring(DB, lines)
	return nil
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
	attributes := strings.Split(DB.global_attributes[LineNo], "||")
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
	return DB.global_values[lineno]
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
	path := DB.global_paths[lineno]
	pathparts := strings.Split(path, "/")
	return pathparts[len(pathparts)-1]
}
func UpdateNodevalue(DB *Database, nodeId int, new_value string) ([]int, error) {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-UpdateNodevalue\n")
	}

	if strings.Contains(new_value, "<") || strings.Contains(new_value, ">") {
		fmt.Printf("Error :Value contains xml\n")

		return []int{}, errors.New("Value contains xml")
	}
	nodes, err := update_nodevalue(DB, nodeId, ReplacewithHTMLSpecialEntities(new_value))

	return nodes, err
}

func update_nodevalue(DB *Database, nodeId int, new_value string) ([]int, error) {
	if (NodeEnd(DB, nodeId) - NodeLine(DB, nodeId)) > 1 {
		fmt.Printf("Error :Cannot update value- Node contains subnodes\n")
		return []int{}, errors.New("Cannot update value- Node contains subnodes")
	}
	content := GetNodeContents(DB, nodeId)
	if len(content) == 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		return []int{}, errors.New("node  doesnot exist")
	}
	value := GetNodeValue(DB, nodeId)
	result := ""
	if len(value) == 0 {
		if strings.Contains(content, "/>") {
			result = content[0:len(content)-3] + ">" + new_value + "</" + GetNodeName(DB, nodeId) + ">"

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
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-UpdateAttributevalue\n")
	}

	content := GetNodeContents(DB, nodeId)
	if len(content) == 0 {
		fmt.Printf("Warning :node  doesnot exist\n")

		return []int{}, errors.New("node  doesnot exist")
	}
	contentparts := strings.Split(content, ">")
	contentparts0 := contentparts[0]
	if strings.Contains(contentparts[0], label) {
		oldvalue := GetNodeAttribute(DB, nodeId, label)
		if DB.Debug_enabled {
			fmt.Printf("replacing -%s -by- %s", label+"=\""+oldvalue+"\"", label+"=\""+value+"\"")
		}
		contentparts0 = strings.ReplaceAll(contentparts0, label+"=\""+oldvalue+"\"", label+"=\""+value+"\"")
	} else {
		contentparts0 = (contentparts0 + " " + label + "=\"" + value + "\"")
	}
	contentnew := contentparts0 + ">"
	for i, part := range contentparts {
		if i > 0 && len(strings.TrimSpace(part)) > 0 {
			contentnew = contentnew + part + ">"
		}
	}
	replacednodes, err := replaceNodeRetainid(DB, nodeId, contentnew)
	if DB.Debug_enabled {
		fmt.Printf("UpdateNodevalue :Updating node %d\n", nodeId)
		fmt.Printf("%s\n", GetNodeContents(DB, nodeId))
	}

	return replacednodes, err
}
func GetNodeContents(DB *Database, nodeId int) string {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-GetNodeContents\n")
	}

	Output := ""
	beginning := NodeLine(DB, nodeId)
	if beginning < 0 {
		return Output
	}
	end := NodeEnd(DB, nodeId)
	if DB.Debug_enabled {
		fmt.Printf("getNodeContents :Fetching Contents from line %d to %d \n", beginning, end)
	}
	var lines []string
	if (end - beginning) > 200 {
		fmt.Printf("\n No of lines more than 200 \n ")
		lines = DB.global_dbLines[beginning : beginning+200]
	} else {
		lines = DB.global_dbLines[beginning:end]
	}

	lines = formatxml(lines)

	for _, line := range lines {
		//line = strings.ReplaceAll(line, "<nil:node>", "")
		//line = strings.ReplaceAll(line, "</nil:node>", "")
		Output = Output + line + "\n"
	}
	if (end - beginning) > 200 {
		Output = Output + "\n .....Remaining lines are not printed......\n "
	}
	return Output
}
func RemoveNode(DB *Database, nodeId int) []int {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-RemoveNode\n")
	}

	nodes := remove_Node(DB, nodeId)

	updateNodenoLineMap(DB, DB.startindex)
	return nodes
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
	for i := startindex; i < end; i++ {
		path := DB.global_paths[startindex]
		path_parts := strings.Split(path, "/")
		hashno := stringtono(DB, path_parts[len(path_parts)-1])
		removeid_fromhashtable(DB, hashno, DB.global_ids[startindex])
		DB.global_dbLines = remove_string(DB.global_dbLines, startindex)
		DB.deleted_ids = append(DB.deleted_ids, DB.global_ids[startindex])
		removedids = append(removedids, DB.global_ids[startindex])
		DB.nodeNoToLineno[DB.global_ids[startindex]] = -1
		DB.global_ids = remove(DB.global_ids, startindex)
		DB.global_paths = remove_string(DB.global_paths, startindex)
		DB.global_values = remove_string(DB.global_values, startindex)
		DB.global_attributes = remove_string(DB.global_attributes, startindex)
	}

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
		return false
	}
	if CommentStarted || xmldeclarationStarted || Comment2Started || CDATAStarted {
		return false
	}
	if len(nodesnames) > 0 {
		return false
	}
	return true
}
func insertAtLine(DB *Database, lineno int, sub_xml string, retainid int) ([]int, error) {
	DB.retainid = retainid
	DB.removeattribute = ""
	DB.pathIdStack_index = 0

	DB.reference_linenotoinsert = lineno - 1

	DB.startindex = lineno
	startindex_tmp := lineno
	path := DB.global_paths[lineno-1]
	if strings.Contains(DB.global_dbLines[lineno-1], "</") || strings.Contains(DB.global_dbLines[lineno-1], "/>") || strings.Contains(DB.global_dbLines[lineno-1], "<!") {
		path_parts := strings.Split(path, "/")
		path = path[0 : len(path)-len(path_parts[len(path_parts)-1])-1]
	}
	DB.path = path
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
	removed := remove_Node(DB, nodeId)
	DB.deleted_ids = remove(DB.deleted_ids, len(DB.deleted_ids)-len(removed))
	nodes, err := insertAtLine(DB, startindex, sub_xml, removed[0])
	return nodes, err
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
	ParentPath := DB.global_paths[parent_nodeLine]

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
	if len(DB.global_paths[requiredline]) >= len(ParentPath) {

		_, _, stat := compare_path(DB.global_paths[requiredline], RequiredPath)

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
	ParentPath := DB.global_paths[parent_nodeLine]
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

			for InsideParent && LineNo < len(DB.global_dbLines) && LineNo <= suspectedLineEnds[index] {
				//fmt.Printf("\nDB.global_paths[LineNo] %s ParentPath %s\n", DB.global_paths[LineNo], ParentPath)
				if isParentPath(ParentPath, DB.global_paths[LineNo]) {

					labels, values, path_matching := compare_path(DB.global_paths[LineNo], QueryPath)

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

										if len(strings.TrimSpace(DB.global_attributes[LineNo])) == 0 {
											all_satisfied = false
										} else {
											attributes := strings.Split(DB.global_attributes[LineNo], "||")
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
											match, _ = regexp.MatchString(valueorAttribute, DB.global_values[LineNo])
										} else {
											valueorAttribute = ReplacewithHTMLSpecialEntities(valueorAttribute)
											match = (valueorAttribute == DB.global_values[LineNo])
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
	if LineNo < 0 {
		return ResultId
	}
	NodePath := DB.global_paths[LineNo]
	parts := strings.Split(NodePath, "/")

	RequiredPath := NodePath[0 : len(NodePath)-len(parts[len(parts)-1])-1]
	return LocateRequireParentdNode(DB, 0, RequiredPath, LineNo)
}

func ChildNodes(DB *Database, nodeId int) []int {
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-ChildNodes\n")
	}
	LineNo := DB.nodeNoToLineno[nodeId]
	var ResultIds []int
	if LineNo < 0 {
		return ResultIds
	}
	NodePath := DB.global_paths[LineNo]
	nodeDepth := len(strings.Split(NodePath, "/"))
	Node_end := DB.nodeNoToLineno[DB.Nodeendlookup[nodeId]] + 1
	InsideParent := true
	LineNo++
	for InsideParent && LineNo < Node_end {
		//fmt.Printf("\npath-%s ", DB.global_paths[LineNo])
		if isParentPath(NodePath, DB.global_paths[LineNo]) {
			if DB.global_paths[LineNo][len(DB.global_paths[LineNo])-2:] == "/~" {

			} else {
				if len(strings.Split(DB.global_paths[LineNo], "/")) == nodeDepth+1 {
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

func GetNode(DB *Database, parent_nodeId int, QUERY_inp string) ([]int, []string) {
	// ldld/dkdicmk/<xe>/kjk[]/lkl/
	for DB.WriteLock {
		fmt.Printf("Waiting for WriteLock-GetNode\n")
	}
	if DB.Debug_enabled {

		fmt.Printf("==Process Query===\n")
		fmt.Printf("ProcessQuery :QUERY_inp- %s\n", QUERY_inp)
	}
	RequiredPath := QUERY_inp
	if strings.Contains(QUERY_inp, "*") {
		QUERY_Parts := strings.Split(QUERY_inp, "*")
		RequiredPath = QUERY_Parts[0]
	}

	RequiredPath_final := ""
	req_parts := strings.Split(RequiredPath, "]")
	for _, req_part := range req_parts {
		String_Value := strings.Split(req_part, "[")
		RequiredPath_final = RequiredPath_final + String_Value[0]
	}
	RequiredPath = RequiredPath_final

	finalPath := ""
	finalPath_parts := strings.Split(QUERY_inp, "]")
	for _, path_part := range finalPath_parts {
		String_Value := strings.Split(path_part, "[")
		finalPath = finalPath + String_Value[0]
	}
	finalPath = strings.ReplaceAll(finalPath, "*", "")
	//fmt.Printf("ProcessQuery :finalPath- %s\n", finalPath)
	QUERY := strings.ReplaceAll(QUERY_inp, "*", "")
	if DB.Debug_enabled {
		fmt.Printf("ProcessQuery :QUERY- %s\n", QUERY)
		fmt.Printf("ProcessQuery :RequiredPath- %s\n", RequiredPath)

	}
	parts := strings.Split(QUERY, "]")
	var labels_result []string
	var final_nodesLineNo []int
	parent_nodeLine := NodeLine(DB, parent_nodeId)
	if parent_nodeLine < 0 {
		return []int{}, []string{}
	}
	final_nodesLineNo = append(final_nodesLineNo, parent_nodeLine)
	labels_result = append(labels_result, "")
	for _, part := range parts {
		String_Value := strings.Split(part, "[")
		QUERYSTR := part
		RegExp := ""
		if len(String_Value) > 1 {
			QUERYSTR = String_Value[0]
			RegExp = String_Value[1]
		}

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
			ResultId := LocateRequireParentdNode(DB, parent_nodeLine, DB.global_paths[parent_nodeLine]+"/"+RequiredPathN, nodeLine)
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
