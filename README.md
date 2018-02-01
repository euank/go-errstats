# go-errstats

```
<montgc> I wonder what percentage of all Go code is "if err != nil"
```

It's fairly common to hear complaints that Go's error handling is verbose, unwieldy, verbose, and verbose.
I don't want to get into that argument here, other than to supply data.

## What does this do?

This is a simple program to parse your Go programs (caveat: will only work if
your program already compiles, might not play nice with build flags) and figure
out what percent of your code is `if err != nil` conditionals.

## What doesn't it work on?

Interesting tidbit, when you do `err := recover(); err != nil`, the type of
`err` there is `interface{}`, so this program doesn't catch that one.

Most importantly, this program doesn't currently handle compound conditionals (e.g. `if err != nil && foo != bar`) and is thus liable to undercount.

It also doesn't handle build tags nor any other such fancery.

If you know of other issues or want to fix any of these, issues and pull requests are quite welcome.

##  Sample outputs

I arbitrarily picked a few packages I had lying around to get some sample output:

### Go standard library

```
$ go version
go version go1.6.1 linux/amd64
$ cd $GOROOT && errstats $(go list ./src/... | grep -v "builtin")
Statistics about your go files:
	Total lines: 	461394
	Total meaningful lines: 	275899
	Total expressions: 	1781676
	Total conditionals: 	36386
	Total conditionals that were error checks: 	2575

	Percent lines that were errchecks: 	0.9333125527819962
	Percent expressions that were errchecks: 	0.14452683877427772
	Percent conditionals that were errchecks: 	7.076897707909636
	Percent of err != nil checks using the var 'err': 	97.32038834951456
```

### rkt

```
$ git checkout v1.29.0
$ errstats $(go list github.com/rkt/rkt/... | grep -v "^github.com/rkt/rkt/vendor/")      
Statistics about your go files:
	Total lines: 	45958
	Total meaningful lines: 	25017
	Total expressions: 	169199
	Total conditionals: 	3837
	Total conditionals that were error checks: 	1923

	Percent lines that were errchecks: 	7.686772994363833
	Percent expressions that were errchecks: 	1.1365315397845142
	Percent conditionals that were errchecks: 	50.11727912431587
	Percent of err != nil checks using the var 'err': 	98.54394175767031
```

### Docker

```
$ git rev-parse --short HEAD
312c826
$ errstats $(sort <(go list ./...) <(go list ./vendor/... ./contrib/...) | uniq -u | grep -v "integration-cli")
Statistics about your go files:
  Total lines:  72028
  Total meaningful lines:   40578
  Total expressions:  276800
  Total conditionals:   6427
  Total conditionals that were error checks:  2781

  Percent lines that were errchecks:  6.85346739612598
  Percent expressions that were errchecks:  1.0046965317919074
  Percent conditionals that were errchecks:   43.27057725221721
  Percent of err != nil checks using the var 'err':   98.92125134843582
```

## Logrus
```
$ git rev-parse --short HEAD
446d1c1
$ errstats $(go list github.com/Sirupsen/logrus/...)
Statistics about your go files:
  Total lines:  1277
  Total meaningful lines:   661
  Total expressions:  4768
  Total conditionals:   68
  Total conditionals that were error checks:  9

  Percent lines that were errchecks:  1.361573373676248
  Percent expressions that were errchecks:  0.18875838926174499
  Percent conditionals that were errchecks:   13.23529411764706
  Percent of err != nil checks using the var 'err':   100
```

## Martini
```
$ git rev-parse --short HEAD
15a4762
$ errstats $(go list github.com/go-martini/martini/...)
Statistics about your go files:
  Total lines:  1088
  Total meaningful lines:   595
  Total expressions:  4335
  Total conditionals:   67
  Total conditionals that were error checks:  8

  Percent lines that were errchecks:  1.3445378151260505
  Percent expressions that were errchecks:  0.1845444059976932
  Percent conditionals that were errchecks:   11.940298507462686
  Percent of err != nil checks using the var 'err':   100
```
## http2
```
$ git rev-parse --short HEAD
6c89489
$ errstats $(go list golang.org/x/net/http2/...)
Statistics about your go files:
  Total lines:  8072
  Total meaningful lines:   4652
  Total expressions:  29802
  Total conditionals:   636
  Total conditionals that were error checks:  79

  Percent lines that were errchecks:  1.6981943250214964
  Percent expressions that were errchecks:  0.2650828803436011
  Percent conditionals that were errchecks:   12.421383647798741
  Percent of err != nil checks using the var 'err':   98.73417721518987
```

## ecs-agent
```
$ git checkout v1.7.0
$ cat <(. ./scripts/shared_env; errstats $(go list github.com/aws/amazon-ecs-agent/agent/...))
Statistics about your go files:
  Total lines:  19539
  Total meaningful lines:   9346
  Total expressions:  64451
  Total conditionals:   807
  Total conditionals that were error checks:  217

  Percent lines that were errchecks:  2.3218489193237746
  Percent expressions that were errchecks:  0.33668988844238257
  Percent conditionals that were errchecks:   26.889714993804215
  Percent of err != nil checks using the var 'err':   97.6958525345622
```

## kubernetes
```
$ git rev-parse --short HEAD
0db3ca4
$ errstats $(go list k8s.io/kubernetes/...)
Statistics about your go files:
	Total lines: 	629444
	Total meaningful lines: 	389486
	Total expressions: 	2489975
	Total conditionals: 	66755
	Total conditionals that were error checks: 	9294

	Percent lines that were errchecks: 	2.3862218410931333
	Percent expressions that were errchecks: 	0.3732567596060201
	Percent conditionals that were errchecks: 	13.92255261778144
	Percent of err != nil checks using the var 'err': 	98.87023886378309
```
