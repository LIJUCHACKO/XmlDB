# About
This is a Go package to treat xml as a database. You can access, modify or replace node

For Example see :"https://github.com/LIJUCHACKO/ods2csv_xmlDB"

## Usage

1. Declare Database.

     `var DB *Database = new(Database) `

2. Load Database.

    `Load_db(DB, "example.html")`

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

    `identifiedNodes, _ := GetNode(DB, 0, "head*/title[This is document title]"`

   -  '*identifiedNodes*' contains list of nodes ids(`<head>`) identified under parent node with id '*0*' (`<html> ..</html>`).
   - '\*'  marks the node we require.
   -  [] encloses attribute or value of the node as additional search criteria.

4. Getting Content of identified nodes.
```
    for _,node:=range identifiedNodes {
        fmt.Printf("\n%s",GetNodeContents(DB, node))
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
   identifiedNodes, _ := GetNode(DB, 0, "head/title*")
   for _,node:=range identifiedNodes {
        fmt.Printf("\n%s",GetNodeValue(DB, node))
   }
```
<u>Output</u>
```
This is document title
```
5 Searching using attribute value.

`identifiedNodes, _ := GetNode(DB, 0, "body*[style="123"]/h1")`
