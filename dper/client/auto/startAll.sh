cd ./dper_booter1
bash -c "./start.sh &"
cd ..
cd ./dper_dper1
bash -c "./start.sh &"
cd ..
cd ./dper_dper2
bash -c "./start.sh &"
cd ..
cd ./dper_dper3
bash -c "./start.sh &"
cd ..
cd ./dper_dper4
bash -c "./start.sh &"
cd ..
cd ./dper_dper5
bash -c "./start.sh &"
cd ..
cd ./dper_dper6
bash -c "./start.sh &"
cd ..
sleep 25 
cd ./dper_dper1
bash -c "./run_contract.sh &"
cd ..
cd ./dper_dper2
bash -c "./run_contract.sh &"
cd ..
cd ./dper_dper3
bash -c "./run_contract.sh &"
cd ..
cd ./dper_dper4
bash -c "./run_contract.sh &"
cd ..
cd ./dper_dper5
bash -c "./run_contract.sh &"
cd ..
cd ./dper_dper6
bash -c "./run_contract.sh &"
cd ..
