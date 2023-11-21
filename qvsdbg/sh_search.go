package main

var shScript = `
files=($(ls *.log*))

echo files $files
output_file="output.txt"
echo "" > "$output_file"

run_grep() {
    keyword="$1"
    filename="$2"
        echo "keyword: $keyword file: $filename"
    sleep 3
    grep "$keyword" "$filename" >> "$output_file"
}

for file in "${files[@]}"; do
        echo "the file is $file end"
    run_grep "test" "$file" &
done

# 等待所有后台进程执行完毕
wait

cat "$output_file"

rm "$output_file"
`
