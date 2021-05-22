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
5. Searching using attribute value.

    `identifiedNodes, _ = xmlDB.GetNode(DB, 0, "<x>*[style=\"123\"]/h1")`

6. Updating node

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

7. Recursive search.

     `identifiedNodes, _ = xmlDB.GetNode(DB, 0, "../h1")`

Also see Example  :"https://github.com/LIJUCHACKO/ods2csv". I have used this library to parse ods xml content.


## Working/Software Design

#### INPUT(sample)

```
<!DOCTYPE html>
<html>
   <head><title>This is document title</title></head>	
   <body style="123">
      <h1>This is a heading</h1>
      <p>Hello World!</p>
   </body>	
</html>
```
#### After parsing- global variables are filled as shown

| global_dbLines                         |   global_ids |  global_paths |  global_values         |  global_attributes      |
|----------------------------------------|--------------|---------------|------------------------|-------------------------|
|` <!DOCTYPE html> `                     |  -1          |               |                        |                         |
|` <head> `                              |   0          | /head         |                        |                         |
|` <title>This is document title</title>`|   1          | /head/title   | This is document title |                         |
| ` <\head> `                            |   2          | /head/~       |                        |                         |
| `<body style="123" font="arial">`      |   3          | /body         |                        | style="123"\|\| font="arial"|
| `<h1>This is a heading</h1>  `         |   4          | /body/h1      |                        |                         |
| `<p>Hello World!</p> `                 |   5          | /body/p       | This is a heading      |                         |
| `</body>`                              |   6          | /body/~       | Hello World!           |                         |
|                                        |              |               |                        |                         |

Note:- If a new node is inserted in between, global_id '7' will be assigned to it. global_id will be unique and will be retained till the node is deleted.


#### 'nodeNoToLineno' contains  line no for every global id.

|   index/global_id|  nodeNoToLineno[index]|
|------------------|-----------------------|
|     0            |    1                  |
|     1            |    2                  |
|     2            |    3                  |
|     3            |    4                  |





#### 'Nodeendlookup' contains global_id of the node end.

|   index/global_id|  Nodeendlookup[index]|
|------------------|----------------------|
|     0            |    2                 |
|     1            |    1                 |
|     2            |    2                 |
|     3            |    6                 |
|     4            |    4                 |
|     5            |    5                 |


##### 'pathKeylookup'  is for quick lookup for line nos corresponding to innermost node.

-Say the path is '/head/title' then hash is calculated for 'title'.

-Corresponding to the hash value a list of global_ids ,arranged in the increasing order of lineno, is stored.


| Hash no |  global id list corresponding to the hash |
|---------|-------------------------------------------|
|    0    |                                           |
|    1    |                                           |
|    2    |   [id1,id2,id3,id4]                       |


-Binary search is utilised  to  find ids, under a parent node, from this list.
