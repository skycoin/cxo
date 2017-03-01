
# http address   | tcp address    | note
#                |                |
# 127.0.0.1:6481 | 127.0.0.1:7481 | testing daemon
# 127.0.0.1:6482 | 127.0.0.1:7482 |
# 127.0.0.1:6483 | 127.0.0.1:7483 |

CXOD=./cxod/cxod
CLI=./cli/cli

all: $(CXOD) $(CLI) launch subscribe terminate

.PHONY: launch
launch:
	@echo
	@echo "Launch"
	@echo
	@echo "==== n1 ===="
	nohup ./cxod/cxod -web-interface-port 6481 \
	                  -remote-term             \
	                  -a 127.0.0.1             \
	                  -p 7481                  \
	                  -launch-browser=f        \
	                  -log-level=debug         \
	                  -name n1                 \
	                  -testing > n1.out 2>&1 &
	@echo "==== n2 ===="
	nohup ./cxod/cxod -web-interface-port 6482 \
	                  -remote-term             \
	                  -a 127.0.0.1             \
	                  -p 7482                  \
	                  -launch-browser=f        \
	                  -log-level=debug         \
	                  -name n2 > n2.out 2>&1 &
	@echo "==== n3 ===="
	nohup ./cxod/cxod -web-interface-port 6483 \
	                  -remote-term             \
	                  -a 127.0.0.1             \
	                  -p 7483                  \
	                  -launch-browser=f        \
	                  -log-level=debug         \
	                  -name n3 > n3.out 2>&1 &
	sleep 5

.PHONY: subscribe
subscribe: inspect
	sleep 1
	@echo
	@echo "Subscribe"
	@echo
	@echo "==== n3 to n2 ===="
	./cli/cli -a http://127.0.0.1:6483 -e 'add subscription 127.0.0.1:7482'
	@echo "==== n2 to n1 ===="
	./cli/cli -a http://127.0.0.1:6482 -e 'add subscription 127.0.0.1:7481'


.PHONY: waiting
waiting:
	@echo "waiting..."
	sleep 20

.PHONY: terminate
terminate: waiting inspect_again
	@echo
	@echo "Terminate"
	@echo
	@echo "==== n1 ===="
	./cli/cli -a http://127.0.0.1:6481 -e close
	@echo "==== n2 ===="
	./cli/cli -a http://127.0.0.1:6482 -e close
	@echo "==== n3 ===="
	./cli/cli -a http://127.0.0.1:6483 -e close

.PHONY: inspect
inspect inspect_again:
	@echo
	@echo "Inspect"
	@echo
	@echo "==== n1 ===="
	./cli/cli -a http://127.0.0.1:6481 -e stat
	@echo "==== n2 ===="
	./cli/cli -a http://127.0.0.1:6482 -e stat
	@echo "==== n3 ===="
	./cli/cli -a http://127.0.0.1:6483 -e stat

#.PHONY: inspect_again
#inspect_again:
#	@echo
#	@echo "Inspect"
#	@echo
#	@echo "==== n1 ===="
#	./cli/cli -a http://127.0.0.1:6481 -e stat
#	@echo "==== n2 ===="
#	./cli/cli -a http://127.0.0.1:6482 -e stat
#	@echo "==== n3 ===="
#	./cli/cli -a http://127.0.0.1:6483 -e stat

$(CXOD):
	go build -o ./cxod/cxod ./cxod/

$(CLI):
	go build -o ./cli/cli ./cli/
