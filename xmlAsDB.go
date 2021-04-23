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
	filename            string
	removeattribute     string
	global_ids          []int
	global_paths        []string
	global_dbLines      []string
	global_values       []string
	global_attributes   []string
	global_lineUniqueid int
	Debug_enabled       bool
	nodeNoToLineno      [MaxInt]int
	pathKeylookup       [MaxInt][]int
	totaldblines        int
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
	for _, ch := range line {
		total = total + int(ch)
	}
	if total > MaxInt {
		total = total - MaxInt
	}
	return total
}
func ExpectedLinenos(DB *Database, path string) ([]int, []int) {
	pathParts := strings.Split(path, "/")
	var NodeNos []int
	var SearchtillEnd int
	index := len(pathParts) - 1
	for {
		if index <= 0 {
			break
		}
		part := pathParts[index]
		if strings.Contains(part, "<") || strings.Contains(part, "..") {

			SearchtillEnd = 1

		} else {
			NodeNos = DB.pathKeylookup[stringtono(part)]

			SearchtillEnd = 0
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
			LineStarts = append(LineStarts, NodeLine(DB, node))
			if SearchtillEnd == 1 {
				LineEnds = append(LineEnds, NodeEnd(DB, node))
			} else {
				LineEnds = append(LineEnds, NodeLine(DB, node))
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
func get_common(set1 []int, set2 []int) []int {
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
func update_path(DB *Database, line string, path string, NodeId int) string {

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

			}

		} else if Node[0:1] == "<" && Node[len(Node)-1:] == "/" {

			/*add*/
			path = path + "/" + NodeName
			DB.removeattribute = NodeName
			lastattribremoved = false

			DB.pathKeylookup[Node_hash] = append(DB.pathKeylookup[Node_hash], NodeId)

		} else if Node[0:1] == "<" {
			/*add*/
			path = path + "/" + NodeName
			DB.pathKeylookup[Node_hash] = append(DB.pathKeylookup[Node_hash], NodeId)
			if strings.Contains(line, "</"+NodeName+">") {
				DB.removeattribute = NodeName

			}
			lastattribremoved = false
		}
	}
	if lastattribremoved {
		path = path + "/~"
	}
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
						if strings.TrimSpace(part)[0:1] == "<?" {
							if len(newline) > 0 {
								newlines = append(newlines, newline)
							}
							newline = part + ">"

						} else if strings.TrimSpace(part)[0:1] == "<!" {
							if len(newline) > 0 {
								newlines = append(newlines, newline)
							}
							newline = part + ">"

						} else if strings.TrimSpace(part)[0:1] == "<" {
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
	if lineno > DB.totaldblines {
		fmt.Printf("warning :node  doesnot exist\n")
		lineno = -1
	}
	return lineno
}
func NodeEnd(DB *Database, nodeId int) int {
	lineno := DB.nodeNoToLineno[nodeId]
	satisfied := true
	orginalpath := DB.global_paths[lineno]
	if nodeId == 0 {
		return DB.totaldblines
	}
	for satisfied && lineno < DB.totaldblines {
		cur_path := DB.global_paths[lineno]
		if isParentPath(orginalpath, cur_path) {
			if lineno+1 < DB.totaldblines {
				if orginalpath == DB.global_paths[lineno+1] {
					lineno++
					satisfied = false
				} else {
					lineno++
				}
			} else {
				lineno++
			}
		} else {
			satisfied = false
		}
	}

	if lineno < DB.totaldblines {
		orginalparts := strings.Split(orginalpath, "/")
		lastpath := DB.global_paths[lineno]
		parts := strings.Split(lastpath, "/")
		lastPart := parts[len(parts)-1]
		if len(orginalparts) == len(parts) {
			if lastPart == "~" {
				lineno++
			}
		}
	}
	if lineno >= DB.totaldblines {
		fmt.Printf("NodeEnd :Error- lineno-%d >= TOTAL LINES-%d ", lineno, DB.totaldblines)
		lineno = DB.totaldblines
		os.Exit(1)
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
	DB.global_lineUniqueid = 0
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
		path = update_path(DB, line, path, DB.global_lineUniqueid)
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
		DB.global_ids = append(DB.global_ids, DB.global_lineUniqueid)
		DB.global_paths = append(DB.global_paths, path)
		DB.global_lineUniqueid++
		if DB.global_lineUniqueid >= MaxInt {
			fmt.Printf("load_db: global_lineUniqueid>=MaxInt")
			os.Exit(1)
		}
	}
	updateNodenoLineMap(DB, 0)

	if DB.Debug_enabled {
		fmt.Printf("load_db :xml db loaded\n No of nodes-%d\n", DB.global_lineUniqueid)
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

func GetNodeAttribute(DB *Database, nodeid int, label string) string {
	LineNo := DB.nodeNoToLineno[nodeid]
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
func GetNodeValue(DB *Database, nodeid int) string {
	return DB.global_values[DB.nodeNoToLineno[nodeid]]
}
func GetNodeName(DB *Database, nodeid int) string {
	path := DB.global_paths[DB.nodeNoToLineno[nodeid]]
	pathparts := strings.Split(path, "/")
	return pathparts[len(pathparts)-1]
}
func GetNodeContents(DB *Database, nodeId int) string {
	Output := ""
	beginning := NodeLine(DB, nodeId)
	end := NodeEnd(DB, nodeId)
	if DB.Debug_enabled {
		fmt.Printf("getNodeContents :Fetching Contents from line %d to %d ", beginning, end)
	}

	lines := DB.global_dbLines[beginning:end]
	lines = formatxml(lines)
	for _, line := range lines {
		line = strings.ReplaceAll(line, "<nil:node>", "")
		line = strings.ReplaceAll(line, "</nil:node>", "")
		Output = Output + line + "\n"
	}

	return Output
}
func RemoveNode(DB *Database, nodeId int) {
	if DB.Debug_enabled {
		fmt.Printf("removeNode :Removing node %d\n", nodeId)
	}
	startindex := NodeLine(DB, nodeId)
	end := NodeEnd(DB, nodeId)
	for i := startindex; i < end; i++ {
		DB.global_dbLines = remove_string(DB.global_dbLines, startindex)
		DB.global_ids = remove(DB.global_ids, startindex)
		DB.global_paths = remove_string(DB.global_paths, startindex)
		DB.global_values = remove_string(DB.global_values, startindex)
		DB.global_attributes = remove_string(DB.global_attributes, startindex)
	}
}
func InsertAtLine(DB *Database, lineno int, sub_xml string) []int {
	DB.removeattribute = ""
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
	for _, line := range additional_lines {
		path = update_path(DB, line, path, DB.global_lineUniqueid)
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
		DB.global_attributes = append(DB.global_attributes, attribute)
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
		DB.global_ids = insert(DB.global_ids, startindex, DB.global_lineUniqueid)
		DB.global_paths = insert_string(DB.global_paths, startindex, path)
		nodes = append(nodes, DB.global_lineUniqueid)
		DB.global_lineUniqueid++
		if DB.Debug_enabled {
			fmt.Printf("insertatline :Inserting New Node %d\n", DB.global_lineUniqueid)
		}
		startindex++
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
	nodes := InsertAtLine(DB, startindex, sub_xml)
	return nodes
}
func AppendAfterNode(DB *Database, nodeId int, sub_xml string) []int {
	end := NodeEnd(DB, nodeId)
	nodes := InsertAtLine(DB, end, sub_xml)
	return nodes
}
func AppendBeforeNode(DB *Database, nodeId int, sub_xml string) []int {
	start := NodeLine(DB, nodeId)
	nodes := InsertAtLine(DB, start, sub_xml)
	return nodes
}
func LocateRequireParentdNode(DB *Database, parent_nodeLine int, RequiredPath string, LineNo_inp int) int {
	//Search 'required node' backward
	if LineNo_inp < 0 || parent_nodeLine < 0 {
		return -1
	}
	ParentPath := DB.global_paths[parent_nodeLine]
	if DB.Debug_enabled {
		fmt.Printf("#####LocateRequireParentdNode###\n ")
		fmt.Printf("ParentPath- %s\n", ParentPath)
		fmt.Printf("LineNo %d\n", LineNo_inp)
		fmt.Printf("RequiredPath %s\n", RequiredPath)

	}
	Starts, _ := ExpectedLinenos(DB, RequiredPath)
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
		//if DB.global_paths[requiredline][0:len(ParentPath)] == ParentPath {
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

	// LineNo := 0

	// LineNo = parent_nodeLine
	parent_endline := NodeEnd(DB, DB.global_ids[parent_nodeLine])
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

	Starts, Ends := ExpectedLinenos(DB, QueryPath)

	for index, start := range Starts {
		//fmt.Printf("\nstart %d start_fin %d\n", start, start_fin)
		if start >= parent_nodeLine && start <= parent_endline {
			LineNo := start
			//fmt.Printf("\nstart %d end %d\n", start, Ends[index])
			for InsideParent && LineNo < len(DB.global_paths) && LineNo <= Ends[index] {

				if isParentPath(ParentPath, DB.global_paths[LineNo]) {

					labels, values, path_matching := compare_path(DB.global_paths[LineNo], QueryPath)

					labelValueStr := ""

					for index, label := range labels {
						labelValueStr = labelValueStr + label + "=" + values[index] + ";"
					}
					if path_matching {
						if onlypath {
							ResultIds = append(ResultIds, LineNo)
							if DB.Debug_enabled {
								fmt.Printf(" QueryPath matching -lineno %d\n", LineNo)
							}
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
	//LineNo := NodeLine(DB, nodeId)
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
	//LineNo := NodeLine(DB, nodeId)
	LineNo := DB.nodeNoToLineno[nodeId]
	var ResultIds []int
	if LineNo < 0 {
		return ResultIds
	}
	NodePath := DB.global_paths[LineNo]
	nodeDepth := len(strings.Split(NodePath, "/"))
	Node_end := NodeEnd(DB, nodeId)
	InsideParent := true
	LineNo++
	for InsideParent && LineNo < Node_end {
		if isParentPath(NodePath, DB.global_paths[LineNo]) { //len(DB.global_paths[LineNo]) >= len(NodePath) {
			//if DB.global_paths[LineNo][0:len(NodePath)] == NodePath {
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

				for _, label := range labels {
					//fmt.Printf("ProcessQuery :label %s\n", label)
					nextlabels = append(nextlabels, label+labels_result[ind])
				}
				for _, identifiedNode := range identifiedNodes {
					nextnodesLineNo = append(nextnodesLineNo, identifiedNode)
					if DB.Debug_enabled {
						fmt.Printf("ProcessQuery :node Line- %d\n", identifiedNode)
					}
				}

			}
			labels_result = nextlabels
			final_nodesLineNo = nextnodesLineNo
		}
	}
	for _, label_res := range labels_result {
		//fmt.Printf("ProcessQuery :label_res %s\n", label_res)
		entries := strings.Split(label_res, ";")
		for _, entry := range entries {
			parts := strings.Split(entry, "=")
			if len(parts[0]) > 0 {
				RequiredPath = strings.ReplaceAll(RequiredPath, parts[0], parts[1])
				//fmt.Printf("ProcessQuery : %s  %s\n", parts[0], parts[1])
			}
		}
	}

	var ResultIds []int
	for _, nodeLine := range final_nodesLineNo {
		if nodeLine > 0 {
			//fmt.Printf("ProcessQuery :parent_nodeLine %s\n", DB.global_paths[parent_nodeLine]+"/"+RequiredPath)
			ResultId := LocateRequireParentdNode(DB, parent_nodeLine, DB.global_paths[parent_nodeLine]+"/"+RequiredPath, nodeLine)
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
