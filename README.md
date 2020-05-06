1. How it run?
>docker-compose up
2. How it test?
>docker exec -i ticker_db mysql -udocker -pdocker <<< "use ticker; select * from ticks;"
