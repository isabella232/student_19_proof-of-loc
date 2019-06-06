# Proof of Location
__EPFL DEDIS Lab: Semester Project 2019 __
__Student:__ Sabrina Kall
__Supervisors:__ Cristina Basescu, Kelong Cong

__Description:__ Implementation of a short-term transaction mechanism based on proof-of-location for intermediary speed-up step during blockchain transactions.

## Content

* __knowthyneighbor__:  implementation
	- __blscosiprotocol__: code for bls collective signature
	- __latencyprotocol__: code for building and testing latency chains
	- __service__: service for running proof-of-location
	- __udp__: bottom-layer messaging infrastructure
*  __python_graphs__: python jupyter notebooks for graphs
	- __var_clusters__
 	- __var_clusters_multiliar__
	- __var_liars__
	- __var_lies__
	- __var_nb_liars__
*  __schemas__: dot files for graphs


## Know-thy-neighbor

### Prerequisites

* [Go](https://golang.org/doc/install) [used for implementation: version 1.2]

### Run

Individual tests can be run from the terminal within the path containing the test's file with the command
```
go test -run NameOfTest
```

If a test requires a more complex command, this will be indicated at the top of the test file

## Generating graphs

### Prerequisites
* [python](https://www.python.org/downloads/) [used for implementation: version 3.7.3 ]
* [Jupyter Notebook](https://jupyter.org/install) 

### Run

Each folder contained in the **python_graphs** directory matches a go test file in the **knowthyneighbor/latencyprotocol** directory, starting with the name **graph_**. When you wish to create a new graph with specific parameters, first run the corresponding test to create the data. It will be written to the __python_graphs__ directory in the matching folder.

To create a graph from this data, do the following steps.

Type into the terminal from the path to **python_graphs** the following command:
```
jupyter notebook
```

This should open a browser with the file system. Navigate to the folder matching your test and open the notebook a file ending in ".ipynb". Make sure the parameters at the top of the notebook match the parameters of the data you created.

Then you can click on the "double arrow" at the top of the notebook to run the code and generate the graph. It will appear both in the notebook, and will be saved in the graphs folder with the same path as the notebook.




