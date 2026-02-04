./detector-debug.exe -p ./test_files --none --secret-marker -v

./detector-debug.exe -p ./test_files --none --layout -v

# 方式1: 手动启用模块 + 规则文件
./detector-debug.exe -p ./test_files --none --hash --hash-rules hash_rules.json -v

# 方式2: 只指定规则文件（自动启用模块）
./detector-debug.exe -p ./test_files --none --hash-rules hash_rules.json -v

 
./detector-debug.exe -p ./test_files --none --electronic -v --stream-rules stream_rules.json -v

./detector-debug.exe -p ./test_files --none --keywords -v

./detector-debug.exe -p ./test_files --all --hash-rules hash_rules.json --stream-rules stream_rules.json -v