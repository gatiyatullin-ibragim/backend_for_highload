@echo off
start "small1" cmd /k ".venv\Scripts\activate && waitress-serve --host=0.0.0.0 --port=8001 --threads=2 loadlab.wsgi:application"
start "small2" cmd /k ".venv\Scripts\activate && waitress-serve --host=0.0.0.0 --port=8002 --threads=2 loadlab.wsgi:application"
start "small3" cmd /k ".venv\Scripts\activate && waitress-serve --host=0.0.0.0 --port=8003 --threads=2 loadlab.wsgi:application"
start "small4" cmd /k ".venv\Scripts\activate && waitress-serve --host=0.0.0.0 --port=8004 --threads=2 loadlab.wsgi:application"