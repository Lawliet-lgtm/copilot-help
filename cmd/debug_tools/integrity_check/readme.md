# 1. 检查指定文件的完整性
./bin/integrity-checker check --file /usr/bin/ls

# 2. 检查自身完整性
./bin/integrity-checker check

# 3. 生成基线哈希（用于保存到配置）
./bin/integrity-checker baseline --file /opt/myapp/server

# 4. 启动持续监控（每 10 秒检查一次）
./bin/integrity-checker watch --file /opt/myapp/server --interval 10s

# 5. 详细模式
./bin/integrity-checker watch --file /opt/myapp/server --interval 5s --verbose

# 6. 查看帮助
./bin/integrity-checker --help
./bin/integrity-checker watch --help

aaa