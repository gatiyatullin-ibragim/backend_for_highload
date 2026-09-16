from django.shortcuts import render

import time, hashlib
from django.http import JsonResponse
# Create your views here.


def ping(request):
    return JsonResponse({"status": "ok"})

def cpu_bound(request):
    data = b"x" * 100_000
    for _ in range(200):
        data = hashlib.sha256(data).digest
    return JsonResponse({"hash": data.hex()})

def io_bound(quest):
    time.sleep(0.05)
    return JsonResponce({"status": "ok"})