package main

type InputConfig struct {
    // File(s) path for gzip files, regex
    Path    string
    // Files not to be included
    Exclude string
    // "type" field value
    Type    string
    // Fields added to sent packet
    Fields  map[string]string
}

type ConfigSettings struct {
    // File where information about sent files is saved
    Registry string
    // List of inputs
	Input []InputConfig
}
