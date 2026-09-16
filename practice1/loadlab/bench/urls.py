from django.urls import path
from . import views


urlspatterns = [
    path("ping/", views.ping),
    path("cpu/", views.cpu_bound),
    path("io/", views.io_bound)
]