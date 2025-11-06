#!/bin/bash
# 测试脚本：手动触发视频详情获取

echo "=== 测试视频详情获取 ==="

# 获取一个没有Pages的视频
VIDEO_ID=44

echo "1. 检查视频 $VIDEO_ID 的Pages..."
curl -s "http://localhost:8080/api/videos/$VIDEO_ID/pages" | head -c 200
echo ""

echo ""
echo "2. 检查视频的tags（用于判断是否已获取详情）..."
curl -s "http://localhost:8080/api/videos/$VIDEO_ID" | grep -o '"tags":[^,}]*'
echo ""

echo ""
echo "3. 触发同步任务..."
curl -s -X POST http://localhost:8080/api/scheduler/trigger
echo ""

echo ""
echo "等待35秒让工作流完成..."
sleep 35

echo ""
echo "4. 再次检查Pages..."
curl -s "http://localhost:8080/api/videos/$VIDEO_ID/pages" | head -c 500
echo ""

echo ""
echo "5. 检查任务状态..."
curl -s "http://localhost:8080/api/tasks/stats"
echo ""

echo ""
echo "6. 检查是否有任务..."
curl -s "http://localhost:8080/api/tasks?page=1&page_size=5" | head -c 1000
echo ""
