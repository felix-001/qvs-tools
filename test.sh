int=1
while(( $int<=60 ))
do
    go run talk-test.go
    let "int++"
    sleep 1
done
