package main

import (
	//"bufio"
	"fmt"

	//"strconv"
	//"strings"

	"github.com/LIJUCHACKO/XmlDB"
)

func main() {

	var DB *xmlDB.Database = new(xmlDB.Database)
	DB.Debug_enabled = false
	xmlDB.Load_db(DB, "sample.html")
	fmt.Printf("\n###Identifying Nodes using Query##")
	identifiedNodes, _ := xmlDB.GetNode(DB, 0, "head*/title[This is document title]")
	for _, node := range identifiedNodes {
		fmt.Printf("\n%s", xmlDB.GetNodeContents(DB, node))
	}
	fmt.Printf("\n###Getting Value of identified nodes.##")
	identifiedNodes, _ = xmlDB.GetNode(DB, 0, "head/title*")
	for _, node := range identifiedNodes {
		fmt.Printf("\n%s", xmlDB.GetNodeValue(DB, node))
	}
	fmt.Printf("\n###Searching using attribute value.##")
	identifiedNodes, _ = xmlDB.GetNode(DB, 0, "body*[style=\"123\"]/h1")
	for _, node := range identifiedNodes {
		fmt.Printf("\n%s", xmlDB.GetNodeContents(DB, node))
	}
	fmt.Printf("\n###[2]Searching using attribute value.##")
	identifiedNodes, _ = xmlDB.GetNode(DB, 0, "<x>*[style=\"123\"]/h1")
	for _, node := range identifiedNodes {
		fmt.Printf("\n%s", xmlDB.GetNodeContents(DB, node))
	}

	fmt.Printf("\n### Updating node value ##\n")
	identifiedNodes, _ = xmlDB.GetNode(DB, 0, "head/title")

	for _, node := range identifiedNodes {
		xmlDB.UpdateNodevalue(DB, node, "test_new")
		fmt.Printf("After updating value\n")
		fmt.Printf("\n%s", xmlDB.GetNodeContents(DB, node))
	}
	identifiedNodes, _ = xmlDB.GetNode(DB, 0, "head")
	ids := xmlDB.ChildNodes(DB, identifiedNodes[0])
	fmt.Printf("\nchildren")
	for _, id := range ids {
		fmt.Printf("\n%s", xmlDB.GetNodeContents(DB, id))
	}
	fmt.Printf("\n### Updating node ##\n")
	identifiedNodes, _ = xmlDB.GetNode(DB, 0, "head/title")
	for _, node := range identifiedNodes {
		newnodes, _ := xmlDB.ReplaceNode(DB, node, "<title>test</title>")
		fmt.Printf("After updation\n")
		fmt.Printf("old node value- %s", xmlDB.GetNodeValue(DB, node)) //no output, existing id is removed and new id added
		fmt.Printf("\nnew node value- %s", xmlDB.GetNodeValue(DB, newnodes[0]))
	}

	fmt.Printf("\n### Updating node attribute##\n")
	identifiedNodes, _ = xmlDB.GetNode(DB, 0, "<x>*[style=\"123\"]/h1")
	for _, node := range identifiedNodes {
		fmt.Printf("\n%s", xmlDB.GetNodeAttribute(DB, node, "style"))
		xmlDB.UpdateAttributevalue(DB, node, "style", "test2")
		fmt.Printf("\nafter updating Attribute-\n%s", xmlDB.GetNodeContents(DB, node))
		xmlDB.UpdateAttributevalue(DB, node, "label", "value")
		fmt.Printf("\nafter adding Attribute-\n%s", xmlDB.GetNodeContents(DB, node))

	}
	xmlDB.SaveAs_DB(DB, "sample_mod.html")

}
