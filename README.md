# parser
An energy saving/efficient modern data protocol format parser.Supported modern data protocol formats are JSON/YAML, etc

Data depth 5000 layers of json;

|  mode  | Benchmark name                |       (1) |             (2) |          (3) |             (4) |
| ------ | ------------------------------| ---------:| ---------------:| ------------:| ---------------:|
| decode | BenchmarkParser        		 |     25206 |    248232 ns/op |       0 B/op |     0 allocs/op |
| decode | BenchmarkParser_Sync       	 |      7780 |    783551 ns/op |   	   0 B/op |   	0 allocs/op |
| decode | BenchmarkJsoniter         	 |     19106 |    315373 ns/op |  366429 B/op |  3124 allocs/op |
| decode | BenchmarkStd        			 |     27234 |    220422 ns/op |  374866 B/op |  2099 allocs/op |