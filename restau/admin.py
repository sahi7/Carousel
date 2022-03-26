from django.contrib import admin
from django.contrib.auth.admin import UserAdmin
from .models import User, Waiter, Manager, Owner
from .models import Customer, Receptionist, Chef

# Register your models here.
admin.site.register(User)
admin.site.register(Customer)
admin.site.register(Receptionist)
admin.site.register(Chef)
admin.site.register(Waiter)
admin.site.register(Manager)
admin.site.register(Owner)