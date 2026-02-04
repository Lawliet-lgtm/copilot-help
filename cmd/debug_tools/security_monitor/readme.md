# 1. 查看配置帮助
./security-monitor config

# 2. 启动所有模块（默认配置）
./security-monitor start --all

# 3. 仅启动完整性校验
./security-monitor start --enable-integrity \
  --integrity-file /tmp/test.bin \
  --integrity-interval 10s

# 4. 仅启动网络监控（监控指定进程）
./security-monitor start --enable-netguard \
  --netguard-pid 12345 \
  --netguard-interval 3s \
  --dry-run

# 5. 完整配置
./security-monitor start \
  --enable-integrity --integrity-file /opt/app/server --integrity-interval 30s \
  --enable-netguard --netguard-pid 12345 --netguard-interval 5s \
  --netguard-whitelist 192.168.0.0/16 \
  --dry-run --verbose
  aaa