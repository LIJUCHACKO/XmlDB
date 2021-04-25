package xmlDB

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const MaxInt = 483647

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		line = strings.ReplaceAll(line, "<nil:node>", "")
		line = strings.ReplaceAll(line, "</nil:node>", "")
		fmt.Fprintln(w, line+"\r")
	}
	return w.Flush()
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

type Database struct {
	filename                string
	removeattribute         string
	global_ids              []int
	deleted_ids             []int
	global_paths            []string
	global_dbLines          []string
	global_values           []string
	global_attributes       []string
	global_lineLastUniqueid int
	Debug_enabled           bool
	nodeNoToLineno          [MaxInt]int
	pathKeylookup           [MaxInt][]int
	Nodeendlookup           [MaxInt]int
	totaldblines            int
	pathIdStack             []int
}

func updateNodenoLineMap(DB *Database, fromLine int) {
	DB.totaldblines = len(DB.global_dbLines)
	lineno := fromLine
	for {
		if lineno >= DB.totaldblines {
			break
		}
		id := DB.global_ids[lineno]
		if id >= 0 {
			DB.nodeNoToLineno[id] = lineno
		}

		lineno++
	}
}
func stringtono(line string) int {
	total := 0
	for i, ch := range line {
		total = total + int(ch)*i
	}
	if total > MaxInt {
		total = total - MaxInt
	}
	return total
}
func suspectedLinenos(DB *Database, path string) ([]int, []int) {
	pathParts := strings.Split(path, "/")
	var NodeNos []int
	SearchtillEnd := 0
	index := len(pathParts) - 1
	for {
		if index <= 0 {
			break
		}
		part := pathParts[index]
		//fmt.Printf("\nkeypart %s", part)
		if strings.Contains(part, "<") || strings.Contains(part, "..") {

			SearchtillEnd = 1

		} else {
			NodeNos = DB.pathKeylookup[stringtono(part)]

			break
		}
		index--
	}
	var LineStarts []int
	var LineEnds []int
	if len(NodeNos) == 0 {
		LineStarts = []int{0}
		LineEnds = []int{DB.totaldblines}
	} else {
		for _, node := range NodeNos {
			LineStarts = append(LineStarts, DB.nodeNoToLineno[node])
			if SearchtillEnd == 1 {
				LineEnds = append(LineEnds, DB.nodeNoToLineno[DB.Nodeendlookup[node]]+1)
			} else {
				LineEnds = append(LineEnds, DB.nodeNoToLineno[node])
			}
		}
	}

	return LineStarts, LineEnds
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
		for _, element2 := range set1 {
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
func update_path(DB *Database, line string, path string, nodeId int) string {

	if len(path) > 3 {
		if path[len(path)-2:len(path)] == "/~" {
			path = path[0 : len(path)-2]
			DB.removeattribute = ""
		}
	}
	lastattribremoved := false
	line = strings.TrimSpace(line)
	Node := ""
	NodeName := ""
	parts := strings.Split(line, ">")
	if len(parts) > 0 {
		parts0 := parts[0]
		parts0 = strings.TrimSpace(parts0)
		Node = parts0
		NodeName = strings.Split(parts0, " ")[0]

	}

	NodeName = strings.Replace(NodeName, "</", "", -1)
	NodeName = strings.Replace(NodeName, "<", "", -1)
	NodeName = strings.Replace(NodeName, "/>", "", -1)
	NodeName = strings.Replace(NodeName, ">", "", -1)
	NodeName = strings.Replace(NodeName, "/", "", -1)

	if len(DB.removeattribute) > 0 {
		if path[len(path)-len(DB.removeattribute):] == DB.removeattribute {
			path = path[0 : len(path)-len(DB.removeattribute)-1]
			DB.removeattribute = ""
		}
	}
	Node_hash := stringtono(NodeName)
	if len(Node) > 1 {
		if Node[0:2] == "</" {
			/*remove*/
			if path[len(path)-len(NodeName):] == NodeName {
				path = path[0 : len(path)-len(NodeName)-1]
				lastattribremoved = true
				len_DB_pathIdStack := len(DB.pathIdStack)
				//fmt.Printf(" nodeid-%d  endid- %d\n", DB.pathIdStack[len_DB_pathIdStack-1], nodeId)
				DB.Nodeendlookup[DB.pathIdStack[len_DB_pathIdStack-1]] = nodeId
				DB.pathIdStack = DB.pathIdStack[0 : len_DB_pathIdStack-1]
			}

		} else if Node[0:1] == "<" && Node[len(Node)-1:] == "/" {

			/*add*/
			path = path + "/" + NodeName
			DB.removeattribute = NodeName
			DB.Nodeendlookup[nodeId] = nodeId
			//fmt.Printf(" nodeid-%d  endid- %d\n", nodeId, nodeId)
			lastattribremoved = false

			DB.pathKeylookup[Node_hash] = append(DB.pathKeylookup[Node_hash], nodeId)

		} else if Node[0:1] == "<" {
			/*add*/
			path = path + "/" + NodeName
			DB.pathKeylookup[Node_hash] = append(DB.pathKeylookup[Node_hash], nodeId)
			if strings.Contains(line, "</"+NodeName+">") {
				DB.removeattribute = NodeName
				DB.Nodeendlookup[nodeId] = nodeId
				//fmt.Printf(" nodeid-%d  endid- %d\n", nodeId, nodeId)
			} else {
				DB.pathIdStack = append(DB.pathIdStack, nodeId)
			}

			lastattribremoved = false
		}
	}
	if lastattribremoved {
		path = path + "/~"
	}
	//fmt.Printf(" path- %s  line-%s \n", path, line)
	return path
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
		}
	}
	return newlines
}
func splitxmlinLines(lines []string) []string {
	newlines := []string{}
	newline := ""
	for _, line := range lines {
		if len(strings.TrimSpace(line)) > 0 {

			parts := strings.Split(line, ">")

			for i, part := range parts {
				part = strings.TrimSpace(part)
				if len(strings.TrimSpace(part)) > 0 {
					if i < len(parts)-1 {
						if strings.TrimSpace(part)[0:1] == "<" {
							if len(newline) > 0 {
								newlines = append(newlines, newline)
							}
							newline = part + ">"

						} else {
							if newline[len(newline)-2:] == "/>" || strings.Contains(newline, "</") {
								//<a>b</a>  or <a/>

								if len(newline) > 0 {
									newlines = append(newlines, newline)
								}
								subparts := strings.Split(part, "<")
								if len(subparts) == 2 {
									newlines = append(newlines, "<nil:node>"+subparts[0]+"</nil:node>")
									newline = "<" + subparts[1] + ">"
								}
							} else {
								//<a>b<c>jhj</c>
								subparts := strings.Split(part, "<")
								if subparts[1][0:1] == "/" {
									newline = newline + part + ">"
								} else {
									if len(newline) > 0 {
										newlines = append(newlines, newline)
									}
									newlines = append(newlines, "<nil:node>"+subparts[0]+"</nil:node>")
									newline = "<" + subparts[1] + ">"
								}

							}

						}

					} else if i == len(parts)-1 {

						if strings.TrimSpace(part)[0:1] == "<" {
							if len(newline) > 0 {
								newlines = append(newlines, newline)
							}
							newline = part
						} else {
							if len(newline) > 0 {
								newlines = append(newlines, newline)
							}
							newlines = append(newlines, "<nil:node>"+part+"</nil:node>")
							newline = ""
						}

					}
				}
			}
			if len(newline) > 0 {
				newlines = append(newlines, newline)
				newline = ""
			}

		}

	}

	return newlines
}

func NodeLine(DB *Database, nodeId int) int {
	lineno := DB.nodeNoToLineno[nodeId]
	if lineno < 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		lineno = -1
	}
	return lineno
}
func NodeEnd(DB *Database, nodeId int) int {
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
func saveas_db(DB *Database, filename string) {
	err := writeLines(DB.global_dbLines, filename)
	if err != nil {
		fmt.Printf("Cannot save db  : %s\n", err)

	}
}
func save_db(DB *Database) {
	if len(DB.filename) == 0 {
		fmt.Printf("Filename not specified\n")
		return
	}
	err := writeLines(DB.global_dbLines, DB.filename)
	if err != nil {
		fmt.Printf("Cannot save db  : %s\n", err)

	}
}

func Load_dbcontent(DB *Database, content []string) {

	DB.global_dbLines = splitxmlinLines(content)
	DB.global_lineLastUniqueid = 0
	DB.removeattribute = ""
	if DB.Debug_enabled {
		fmt.Printf("load_db :formating over\n")
	}
	path := ""

	for _, line := range DB.global_dbLines {
		if strings.Contains(line, "<?xml") || strings.Contains(line, "<!DOCTYPE") {
			DB.global_values = append(DB.global_values, "")
			DB.global_ids = append(DB.global_ids, -1)
			DB.global_paths = append(DB.global_paths, "")
			DB.global_attributes = append(DB.global_attributes, "")
			continue
		}
		path = update_path(DB, line, path, DB.global_lineLastUniqueid)
		//fmt.Printf("\npath-%s", path)
		Value := ""
		parts := strings.Split(line, ">")
		part0 := strings.TrimSpace(parts[0])
		if part0[len(part0)-1:] == "/" {
			part0 = part0[0 : len(part0)-1]
		}
		part0parts := strings.Split(part0, " ")
		attribute := ""
		if len(part0parts) > 1 {

			ind := 0
			for _, attribute_each := range part0parts {
				if len(attribute_each) > 0 {
					if ind > 0 {
						attribute = attribute + "||" + strings.TrimSpace(attribute_each)
					} else {
						attribute = attribute
					}
					ind++
				}

			}

		}
		DB.global_attributes = append(DB.global_attributes, attribute)
		part1 := strings.TrimSpace(parts[1])
		if len(part1) > 0 {
			if part1[0] == '<' {

			} else {
				parts2 := strings.Split(part1, "<")
				Value = strings.TrimSpace(parts2[0])

			}
		}
		DB.global_values = append(DB.global_values, Value)
		DB.global_ids = append(DB.global_ids, DB.global_lineLastUniqueid)
		DB.global_paths = append(DB.global_paths, path)
		DB.global_lineLastUniqueid++
		if DB.global_lineLastUniqueid >= MaxInt {
			fmt.Printf("load_db: Total no. of Uniqueid>= MaxInt, Please increase MaxInt")
			os.Exit(1)
		}
	}
	updateNodenoLineMap(DB, 0)

	if DB.Debug_enabled {
		fmt.Printf("load_db :xml db loaded\n No of nodes-%d\n", DB.global_lineLastUniqueid)
	}
}
func Load_db(DB *Database, filename string) {
	DB.filename = filename
	lines, err := readLines(filename)
	if err != nil {
		fmt.Printf("Cannot load_db :Read : %s\n", err)

	}
	Load_dbcontent(DB, lines)
}

func GetNodeAttribute(DB *Database, nodeId int, label string) string {
	LineNo := DB.nodeNoToLineno[nodeId]
	if LineNo < 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		return ""
	}
	attributes := strings.Split(DB.global_attributes[LineNo], "||")
	for _, attri := range attributes {

		LabelValue := strings.Split(attri, "=\"")
		if len(LabelValue) >= 2 {
			if LabelValue[0] == label {
				Value := strings.TrimSpace(LabelValue[1])
				//removing end quotes
				//fmt.Printf("Value %s\n", Value)

				return Value[:len(Value)-1]
			}
		}
	}
	return ""
}
func GetNodeValue(DB *Database, nodeId int) string {
	lineno := DB.nodeNoToLineno[nodeId]
	if lineno < 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		return ""
	}
	return DB.global_values[lineno]
}
func GetNodeName(DB *Database, nodeId int) string {
	lineno := DB.nodeNoToLineno[nodeId]
	if lineno < 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		return ""
	}
	path := DB.global_paths[lineno]
	pathparts := strings.Split(path, "/")
	return pathparts[len(pathparts)-1]
}
func UpdateNodevalue(DB *Database, nodeId int, new_value string) []int {
	content := GetNodeContents(DB, nodeId)
	if len(content) == 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		return []int{}
	}
	value := GetNodeValue(DB, nodeId)
	content = strings.ReplaceAll(content, ">"+value+"<", ">"+strings.TrimSpace(new_value)+"<")
	replacednodes := replaceNodeRetainid(DB, nodeId, content)
	if DB.Debug_enabled {
		fmt.Printf("UpdateNodevalue :Updating node %d\n", nodeId)
		fmt.Printf("%s\n", GetNodeContents(DB, nodeId))
	}
	return replacednodes
}
func UpdateAttributevalue(DB *Database, nodeId int, label string, value string) []int {
	content := GetNodeContents(DB, nodeId)
	if len(content) == 0 {
		fmt.Printf("Warning :node  doesnot exist\n")
		return []int{}
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
		if i > 0 && len(part) > 0 {
			contentnew = contentnew + part + ">"
		}
	}

	replacednodes := replaceNodeRetainid(DB, nodeId, contentnew)
	if DB.Debug_enabled {
		fmt.Printf("UpdateNodevalue :Updating node %d\n", nodeId)
		fmt.Printf("%s\n", GetNodeContents(DB, nodeId))
	}
	return replacednodes
}
func GetNodeContents(DB *Database, nodeId int) string {
	Output := ""
	beginning := NodeLine(DB, nodeId)
	if beginning < 0 {
		return Output
	}
	end := NodeEnd(DB, nodeId)
	if DB.Debug_enabled {
		fmt.Printf("getNodeContents :Fetching Contents from line %d to %d \n", beginning, end)
	}

	lines := DB.global_dbLines[beginning:end]
	lines = formatxml(lines)
	for _, line := range lines {
		//line = strings.ReplaceAll(line, "<nil:node>", "")
		//line = strings.ReplaceAll(line, "</nil:node>", "")
		Output = Output + line + "\n"
	}

	return Output
}
func RemoveNode(DB *Database, nodeId int) []int {
	if DB.Debug_enabled {
		fmt.Printf("removeNode :Removing node %d\n", nodeId)
	}
	startindex := NodeLine(DB, nodeId)
	end := NodeEnd(DB, nodeId)
	var removedids []int
	for i := startindex; i < end; i++ {
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
func insertAtLine(DB *Database, lineno int, sub_xml string, retainid int) []int {
	DB.removeattribute = ""
	DB.pathIdStack = DB.pathIdStack[:0]
	var nodes []int
	startindex := lineno
	startindex_tmp := lineno
	path := DB.global_paths[lineno]
	if strings.Contains(DB.global_dbLines[lineno], "</") || strings.Contains(DB.global_dbLines[lineno], "/>") {
		path_parts := strings.Split(path, "/")
		path = path[0 : len(path)-len(path_parts[len(path_parts)-1])-1]
	}
	newlines := strings.Split(sub_xml, "\n")
	additional_lines := splitxmlinLines(newlines)
	first := true
	for _, line := range additional_lines {
		unique_id := DB.global_lineLastUniqueid
		if retainid > 0 && first {
			unique_id = retainid
			first = false
		} else {
			if DB.global_lineLastUniqueid >= MaxInt {
				if len(DB.deleted_ids) > 0 {
					unique_id = DB.deleted_ids[0]
					DB.deleted_ids = DB.deleted_ids[1:]
				} else {
					fmt.Printf("InsertAtLine: Total no. of Uniqueid>= MaxInt, Please increase MaxInt")
					os.Exit(1)
				}
			}
		}

		path = update_path(DB, line, path, unique_id)
		if DB.Debug_enabled {
			fmt.Printf("insertatline :Inserting %s  %s\n", line, path)
		}

		Value := ""
		parts := strings.Split(line, ">")
		part0 := strings.TrimSpace(parts[0])
		if part0[len(part0)-1:] == "/" {
			part0 = part0[0 : len(part0)-1]
		}
		part0parts := strings.Split(part0, " ")
		attribute := ""
		if len(part0parts) > 1 {
			ind := 0
			for _, attribute_each := range part0parts {
				if len(attribute_each) > 0 {
					if ind > 0 {
						attribute = attribute + "||" + strings.TrimSpace(attribute_each)
					}
					ind++
				}

			}
		}

		DB.global_attributes = insert_string(DB.global_attributes, startindex, attribute)
		part1 := strings.TrimSpace(parts[1])
		if len(part1) > 0 {
			if part1[0] == '<' {

			} else {
				parts2 := strings.Split(part1, "<")
				Value = strings.TrimSpace(parts2[0])
			}
		}
		DB.global_dbLines = insert_string(DB.global_dbLines, startindex, line)
		DB.global_values = insert_string(DB.global_values, startindex, Value)
		DB.global_ids = insert(DB.global_ids, startindex, unique_id)
		DB.global_paths = insert_string(DB.global_paths, startindex, path)
		nodes = append(nodes, unique_id)

		if DB.Debug_enabled {
			fmt.Printf("insertatline :Inserting New Node %d\n", unique_id)
		}
		startindex++

		if DB.global_lineLastUniqueid < MaxInt && unique_id == DB.global_lineLastUniqueid {
			DB.global_lineLastUniqueid++
		}

	}
	updateNodenoLineMap(DB, startindex_tmp-1)

	return nodes
}
func ReplaceNode(DB *Database, nodeId int, sub_xml string) []int {
	if DB.Debug_enabled {
		fmt.Printf("replaceNode :Replacing node %d\n", nodeId)
	}
	startindex := NodeLine(DB, nodeId)
	RemoveNode(DB, nodeId)
	nodes := insertAtLine(DB, startindex, sub_xml, -1)
	return nodes
}
func replaceNodeRetainid(DB *Database, nodeId int, sub_xml string) []int {
	if DB.Debug_enabled {
		fmt.Printf("replaceNodeRetainid :Replacing node %d\n", nodeId)
	}
	startindex := NodeLine(DB, nodeId)
	removed := RemoveNode(DB, nodeId)
	DB.deleted_ids = remove(DB.deleted_ids, len(DB.deleted_ids)-len(removed))
	nodes := insertAtLine(DB, startindex, sub_xml, removed[0])
	return nodes
}
func AppendAfterNode(DB *Database, nodeId int, sub_xml string) []int {
	end := NodeEnd(DB, nodeId)
	nodes := insertAtLine(DB, end, sub_xml, -1)
	return nodes
}
func AppendBeforeNode(DB *Database, nodeId int, sub_xml string) []int {
	start := NodeLine(DB, nodeId)
	nodes := insertAtLine(DB, start, sub_xml, -1)
	return nodes
}
func LocateRequireParentdNode(DB *Database, parent_nodeLine int, RequiredPath string, LineNo_inp int) int {
	//Search 'required node' backward
	if LineNo_inp < 0 || parent_nodeLine < 0 {
		return -1
	}
	ParentPath := DB.global_paths[parent_nodeLine]
	Starts, _ := suspectedLinenos(DB, RequiredPath)

	if DB.Debug_enabled {
		fmt.Printf("#####LocateRequireParentdNode###\n ")
		fmt.Printf("ParentPath- %s\n", ParentPath)
		fmt.Printf("LineNo %d\n", LineNo_inp)
		fmt.Printf("RequiredPath %s\n", RequiredPath)
		fmt.Printf("parent_nodeLine %d\n", parent_nodeLine)
		fmt.Printf("No of Suspected lines-%d\n", len(Starts))
	}

	//locate line just above LineNo_inp
	requiredline := 0
	for _, start := range Starts {
		if start >= parent_nodeLine && start <= LineNo_inp {
			if start > requiredline {
				requiredline = start
			}
		}
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

	Starts, Ends := suspectedLinenos(DB, QueryPath)
	//fmt.Printf("\nlen(start) %d QueryPath %s\n", len(Starts), QueryPath)
	for index, start := range Starts {
		//fmt.Printf("\nstart %d end %d\n", start, Ends[index])
		if start >= parent_nodeLine && start <= parent_endline {
			LineNo := start

			for InsideParent && LineNo < len(DB.global_paths) && LineNo <= Ends[index] {

				if isParentPath(ParentPath, DB.global_paths[LineNo]) {

					labels, values, path_matching := compare_path(DB.global_paths[LineNo], QueryPath)

					labelValueStr := ""

					for index, label := range labels {
						labelValueStr = labelValueStr + label + "=" + values[index] + ";"
					}
					if path_matching {
						//fmt.Printf("\npath matching %s", QueryPath)
						if onlypath {
							ResultIds = append(ResultIds, LineNo)
							if DB.Debug_enabled {
								fmt.Printf(" QueryPath matching -lineno %d\n", LineNo)
							}
							Label_Values = append(Label_Values, labelValueStr)
						} else {
							//iterate through all
							values_attributes := strings.Split(RegExp, ";")
							all_satisfied := true
							for _, valueorAttribute := range values_attributes {

								if len(valueorAttribute) > 0 {

									if strings.Contains(RegExp, "=") {

										if len(strings.TrimSpace(DB.global_attributes[LineNo])) == 0 {
											all_satisfied = false
										} else {
											attributes := strings.Split(DB.global_attributes[LineNo], "||")

											for _, attrib := range attributes {

												if len(attrib) > 0 {
													match := false
													if isRegExp {
														match, _ = regexp.MatchString(valueorAttribute, attrib)
													} else {
														match = (valueorAttribute == attrib)
													}
													if !match {
														all_satisfied = false
													}
												} else {

												}
											}
										}

									} else {
										match := false
										if isRegExp {
											match, _ = regexp.MatchString(valueorAttribute, DB.global_values[LineNo])
										} else {
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
									fmt.Printf("QueryPath matching -lineno %d\n", LineNo)
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
	LineNo := DB.nodeNoToLineno[nodeId]
	ResultId := -1
	if LineNo < 0 {
		return ResultId
	}
	NodePath := DB.global_paths[LineNo]
	parts := strings.Split(NodePath, "/")

	RequiredPath := NodePath[0 : len(NodePath)-len(parts[len(parts)-1])-1]
	//Search 'required node' backward
	InsideParent := true
	for InsideParent {
		if isParentPath(RequiredPath, DB.global_paths[LineNo]) {
			if DB.global_paths[LineNo] == RequiredPath {
				ResultId = DB.global_ids[LineNo]
				return ResultId
			}
		} else {
			InsideParent = false

		}
		LineNo--
	}
	return ResultId
}

func ChildNodes(DB *Database, nodeId int) []int {
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
