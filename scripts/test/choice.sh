./bin/serverledge-cli create -f double --memory 200 --src examples/double.py --runtime python310 --handler double.handler -i input:Int -o input:Int
./bin/serverledge-cli create -f inc --memory 200 --src examples/inc.py --runtime python310 --handler inc.handler -i input:Int -o input:Int
./bin/serverledge-cli create -f hello --memory 200 --src examples/hello.py --runtime python310 --handler hello.handler -i input:Text
./bin/serverledge-cli compose -f choice -s ./internal/test/asl/choice_boolexpr.json
echo "first branch"
./bin/serverledge-cli play -f choice  -j ./scripts/test/choice_1.json
echo "second branch"
./bin/serverledge-cli play -f choice  -j ./scripts/test/choice_2.json
echo "third branch"
./bin/serverledge-cli play -f choice  -j ./scripts/test/choice_3.json -v
./bin/serverledge-cli uncompose -f choice