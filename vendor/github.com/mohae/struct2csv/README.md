# struct2csv
[![Build Status](https://travis-ci.org/mohae/struct2csv.png)](https://travis-ci.org/mohae/struct2csv)

Create either a CSV file or `[][]string` from a slice of structs.

## About
Struct2csv takes struct(s) and transforms them into CSV which can be either written to a provide writer with `encoding/csv` or returned as `[][]string` using `Marshal`.

The field names are used as the first row of the csv data.  Use of field tags for csv header row column names is supported.  By default, struct2csv uses the looks for field tags for `csv`.  It can be configured to use the values of other field tags, e.g. `yaml` or `json`, instead, or, to ignore field tags.  

Each struct in the provided slice becomes its own row.  Slices are represented as a single column, with each element separated by a comma.  Maps are represented as a single column; entries become a comma separated list of key:value pairs.  Embedded struct fields become their own columns unless they are part of a slice or map.  More complex types use separators, "(" and ")" by default, to group subtypes: e.g. a field with the type `map[string][]string` will have an output similar to `key1: (key1, slice, values)`.  

The separators to use can be set with `encoder.SetSeparators(begin, end)`. Passing empty strings, `""`, will result in no separators being used.  The separators are used for composite types with lists.

Only exported fields become part of the csv data, unless the field's struct tag is `-`.  Some types, like channels and funcs, are skipped.

If a non-struct Kind is received, an error will occur. If a non-slice is passed to `Marshal` or `WriteStructs`, an error will be returned.

## Usage
### Using Directly
A new encoder can be created with the `New()` func.  This will return a new encoder initalized with default value.  The encoder can be configured using it's exposed methods.  When using the encoder, all data is returned as `[]string` or `[][]string` values.  It is your responsibility to encode it to CSV using `encoding/csv`.

#### Extract data from a slice of structs:

    data := []MyStruct{MyStruct{}, MyStruct{}}
    enc := struct2csv.New()
    rows, err := enc.Marshal(data)
    if err != nil {
            // handle error
    }

#### Extract data from a slice of structs; one at a time:

    data := []MyStruct{MyStruct{}, MyStruct{}}
    enc := struct2csv.New()
    var rows [][]string
    // get the column names first
    colhdrs, err := enc.GetHeaders(data[0])
    if err != nil {
            // handle error
    }
    rows = append(rows, colhdrs)
    // get the data from each struct
    for _, v := range data {
            row, err := enc.GetRow(v)
            if err != nil {
                    // handle error
            }
            rows = append(rows, row)
    }

### Using with the Writer
A writer can be created by passing an `io.Writer` to `NewWriter(w)` and using it's methods. The Writer wraps encoding/csv's Writer and struct2csv's Encoder.

This Writer exposes csv,Writer's methods using wrapper methods.  Struct2csv's Writer has additional methods for configuring the encoder and working with Structs.

#### Create CSV from a slice of structs:

    data := []MyStruct{MyStruct{}, MyStruct{}}
    buff := &bytes.Buffer{}
    w := struct2csv.NewWriter(buff)
    err := w.WriteStructs(data)
    if err != nil {
            // handle error
    }

#### Extract data from a slice of structs; one at a time:

    data := []MyStruct{MyStruct{}, MyStruct{}}
    buff := &bytes.Buffer{}
    w := struct2csv.NewWriter(buff)
    // set the column names first
    err := w.WriteColNames(data[0])
    if err != nil {
            // handle error
    }
    // get the data from each struct
    for _, v := range data {
            err = w.Write
            if err != nil {
                    // handle error
            }
            rows = append(rows, row)
    }
    // must flush the writer
    w.Flush()
    fmt.Println(buff.String())

### Configuration of an Encoder
By default, an encoder will use tag fields with the tag `csv`, if they exist, as the column header value for a field. If such a tag does not exist, the column name will be used.  The encoder will also use `(` and `)` as its begin and end separator values.

The separator values can be changed with the `Encoder.SetSeparators(beginSep, endSep)` method.  If the separators are set to `""`, an empty string, nothing will be used.  This mainly applies to lists.

The tag that the encoder uses can be changed by calling `Encoder.SetTag(value)`.

Tags can be ignored by calling `Encoder.SetUseTag(false)`.  This will result in the struct field names being used as the colmn header values.

## Supported types
The following `reflect.Kind` are supported:  
```
Bool
Int
Int8
Int16
Int32
Int64
Uint
Uint8
Uint16
Uint32
Uint64
Float32
Float64
Complex64
Complex128
Array
Map
Ptr
Slice
String
Struct
```

The following `reflect.Kind` are not supported, or have not yet been implemented.  Any fields using any of these kinds will be ignored. If a map uses any of these Kinds in either its key or value, it will be ignored.
```
Chan
Func
Uintptr
Interface
UnsafePointer
```

### Embedded types
If a type is embedded, any exported fields within that struct become their own columns with the field name being the column name, unless a field tag has been defined.  The name of the embedded struct does not become part of the column header name.

### Maps, Slices, and Arrays
#### Map
Maps are a single column in the resulting CSV as maps can have a variable number of elements and there is no way to account for this within CSV.  Each map element becomes a `key:value` pair with each element seperated by a `,`.  Keys are sorted.  

    map[string]string{"Douglas Adams": "Hitchhiker's Guide to the Galaxy", "William Gibson": "Neuromancer"}

becomes:

    Douglas Adams:Hitchhiker's Guide to the Galaxy, William Gibson:Neuromancer

If the map's value is a composite type, the values of the composite type become a comma separated list surrounded by `()`.

    map[string][]string{
            "William Gibson": []string{"Neuromancer" ,"Count Zero", "Mona Lisa Overdrive"},
            "Frank Herbert": []string{"Destination Void", "Jesus Incident", "Lazurus Effect"},
    }

becomes:

    Frank Herbert:(Destination Void, Jesus Incident, Lazurus Effect),William Gibson:(Neuromancer, Count Zero, Mona Lisa Overdrive)

#### Slices and Arrays
Slices and arrays are a single column in the resulting CSV as slices can have a variable number of elements and there is no way to account for this within CSV.  Arrays are treated the same as slices.  Slices become a comma separated list of values.

#### Structs
Struct fields become their own column.  If the struct is embedded, only its field name is used for the column name.  This may lead to some ambiguity in column names.  Options to either prefix the embedded struct's field name with the struct name, or with the full path to the struct, in the case of deeply nested embedded structs may be added in the future (pull requests supporting this are also welcome!)  If the struct is part of a composite type, like a map or slice, it will be part of that column with its data nested, using separators as appropriate.

#### Pointers and nils
Pointers are dereferenced.  Struct field types using multiple, consecutive pointers, e.g. `**string`, are not supported.  Struct fields with composite types support mulitple, non-consecutive pointers, for whatever reason, e.g. `*[]*string`, `*map[*string]*[]*string`, are supported.

A nil result in an empty string, regardless of its type.

### Header row
It is possible to get the header row for a struct by calling the `GetHeaders` func with the struct from which you want the column names.  The names are returned as a `[]string`.

### Data of a single struct
It is possible to get the data from a single struct by calling the `GetStructData` func with the struct from which you want the column data. The data is returned as a `[]string`.

## TODO

* Add option to add names of embedded structs to the column header for its fields.
* Add support for `interface{}`
