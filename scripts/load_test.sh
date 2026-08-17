echo "============================================================"
echo "  НАГРУЗОЧНОЕ ТЕСТИРОВАНИЕ DOCSEARCH"
echo "============================================================"

if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "Сервер не запущен. Запусти: ./docsearch.exe serve"
    exit 1
fi

echo "Сервер запущен"
echo ""

go run scripts/load.go