# About
This is a Go package to treat xml as a native database. You can access, modify or replace node


## Usage

1. Declare Database.

     `var DB *xmlDB.Database = new(xmlDB.Database) `

2. Load Database.

    `xmlDB.Load_db(DB, "sample.html")`

<u> Content of example.html</u>

```
<!DOCTYPE html>
<html>
   <head>
      <title>This is document title</title>
   </head>	
   <body style="123">
      <h1>This is a heading</h1>
      <p>Hello World!</p>
   </body>	
</html>
```

3. Identifying Nodes using Query.

    `identifiedNodes, _ := xmlDB.GetNode(DB, 0, "head*/title[This is document title]")`

   -  '*identifiedNodes*' contains list of nodes ids(`<head>`) identified under parent node with id '*0*' (`<html> ..</html>`).
   - '\*'  marks the node we require.
   -  [] encloses attribute or value of the node as additional search criteria.

4. Getting Content of identified nodes.
```
    for _,node:=range identifiedNodes {
        fmt.Printf("\n%s",xmlDB.GetNodeContents(DB, node))
    }
```
<u>Output</u>
```
    <head>
          <title>This is document title</title>
    </head>
```
5. Getting Value of identified nodes.

```
   identifiedNodes, _ := xmlDB.GetNode(DB, 0, "head/title*")
   for _,node:=range identifiedNodes {
        fmt.Printf("\n%s",xmlDB.GetNodeValue(DB, node))
   }
```
<u>Output</u>
```
This is document title
```
5 Searching using attribute value.

`identifiedNodes, _ = xmlDB.GetNode(DB, 0, "<x>*[style=\"123\"]/h1")`

6 Updating node

```
	fmt.Printf("\n### Updating node value##\n")
	identifiedNodes, _ = xmlDB.GetNode(DB, 0, "head/title")
	for _, node := range identifiedNodes {
		newnodes := xmlDB.ReplaceNode(DB, node, "<title>test</title>")
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
```
<u>Output</u>
```
### Updating node value##
After updation
Warning :node  doesnot exist
old node value- 
new node value- test
### Updating node attribute##

123
after updating Attribute-
<body style="test2">
  <h1>This is a heading with style</h1>
  <p>Hello World!</p>
</body>

after adding Attribute-
<body style="test2" label="value">
  <h1>This is a heading with style</h1>
  <p>Hello World!</p>
</body>
```


Also Example see :"https://github.com/LIJUCHACKO/ods2csv_xmlDB". I have used this library to parse ods xml content.