from django.db import models
from django.conf import settings
from django.utils import timezone
from django.contrib.auth import get_user_model
from django.contrib.auth.models import PermissionsMixin
from django.contrib.auth.base_user import AbstractBaseUser
from .managers import UserManager, CustomerManager, ReceptionistManager, ChefManager, WaiterManager, RestManagerManager, OwnerManager


user_model = settings.AUTH_USER_MODEL

class User(AbstractBaseUser, PermissionsMixin):

	######## GENERAL USER FIELDS ##########
	username = models.CharField(max_length=20, unique=True)
	name = models.CharField(max_length=100)
	surname = models.CharField(max_length=100)
	email = models.EmailField(unique=True, db_index=True)
	address = models.CharField(max_length=250)
	phone_number = models.CharField(max_length=14)
	is_online = models.BooleanField(default=False)
	is_admin = models.BooleanField(default=False)
	is_staff = models.BooleanField(default=False)
	is_active = models.BooleanField(default=False)
	is_superuser = models.BooleanField(default=False)

	#CUSTOM USERS
	is_customer = models.BooleanField(default=False)
	is_chef = models.BooleanField(default=False)
	is_waiter = models.BooleanField(default=False)
	is_receptionist = models.BooleanField(default=False)
	is_manager = models.BooleanField(default=False)
	is_owner = models.BooleanField(default=False)

	######## GENERAL USER FIELDS END ##########

	######## CUSTOMER RELATIVE FIELDS ##########
	MALE = 'M'
	FEMALE = 'F'
	GENDER_CHOICES = (
			(MALE, 'Male'),
			(FEMALE, 'Female')
		)
	gender = models.CharField(max_length=2, choices=GENDER_CHOICES, default=MALE, null=True)
	profile_picture = models.ImageField(upload_to='uploads/customer/', null=True)

	######## CUSTOMER RELATIVE FIELDS END ##########

	######## RECEPTIONIST/CHEF/WAITER RELATIVE FIELDS START ##########
	date_of_birth = models.DateField(null=True)

	######## RECEPTIONIST/CHEF/WAITER RELATIVE RELATIVE FIELDS END ##########

	######## MANAGER RELATIVE FIELDS START ##########
	qualification = models.CharField(max_length=600, null=True)

	######## MANAGER RELATIVE FIELDS END ##########

	created_at = models.DateTimeField(auto_now_add=True)

	ACTIVE = 'A'
	BLACKLISTED = 'B'
	CANCELLED = 'C'
	USER_STATUS = (
			(ACTIVE, 'Active'),
			(CANCELLED, 'Cancelled'),
			(BLACKLISTED, 'Blacklisted'),
		)
	user_status = models.CharField(max_length=2, choices=USER_STATUS, default=ACTIVE)

	USERNAME_FIELD = 'username'
	REQUIRED_FIELDS = ['name', 'surname', 'email',]

	objects = UserManager()

	def full_name(self):
		return (self.name + ' ' + self.surname)

	def short_name(self):
		return self.name

	def natural_key(self):
		return (self.name, self.surname)

	def __str__(self):
		return self.username


class Customer(User, PermissionsMixin):

	USERNAME_FIELD = 'username'
	REQUIRED_FIELDS = ['name', 'surname', 'phone_number',]

	objects = CustomerManager()

	def __str__(self):
		return self.name


class Receptionist(User, PermissionsMixin):

	USERNAME_FIELD = 'username'
	REQUIRED_FIELDS = ['name', 'surname', 'date_of_birth', 'email',]

	objects = ReceptionistManager()

	def __str__(self):
		return self.name


class Chef(User, PermissionsMixin):

	USERNAME_FIELD = 'username'
	REQUIRED_FIELDS = ['name', 'surname', 'date_of_birth', 'email',]

	objects = ChefManager()

	def __str__(self):
		return self.name


class Waiter(User, PermissionsMixin):

	USERNAME_FIELD = 'username'
	REQUIRED_FIELDS = ['name', 'surname', 'date_of_birth', 'email',]

	objects = WaiterManager()

	def __str__(self):
		return self.name


class Manager(User, PermissionsMixin):

	USERNAME_FIELD = 'username'
	REQUIRED_FIELDS = ['name', 'surname', 'date_of_birth', 'email',]

	objects = RestManagerManager()

	def __str__(self):
		return self.name



class Owner(User, PermissionsMixin):

	USERNAME_FIELD = 'username'
	REQUIRED_FIELDS = ['name', 'surname', 'email',]

	objects = OwnerManager()

	def __str__(self):
		return self.name



class Restaurant(models.Model):
	name = models.CharField(null=False, max_length=100)
	address = models.CharField(max_length=200)
	details = models.CharField(max_length=600)
	manager = models.ForeignKey(User, null=True, on_delete=models.CASCADE)
	created_at = models.DateTimeField(auto_now_add=True)
	opened_on = models.DateField(null=False)

	class Meta:
		ordering = ('name',)

	def __str__(self):
		return self.name



class Branch(models.Model):
	name = models.CharField(null=False, max_length=100)
	address = models.CharField(max_length=200)
	details = models.CharField(max_length=600)
	restaurant = models.ForeignKey(Restaurant,
		related_name='branches',
		on_delete=models.CASCADE)
	created_at = models.DateTimeField(auto_now_add=True)
	opened_on = models.DateField(null=False)

	class Meta:
		ordering = ('name',)

	def __str__(self):
		return self.name




class Menu(models.Model):
	title = models.CharField(null=False, max_length=100)
	details = models.CharField(max_length=600)
	created_at = models.DateTimeField(auto_now_add=True)
	branch = model.OneToOneField(Branch, on_delete=models.DO_NOTHING)

	class Meta:
		ordering = ('title',)

	def __str__(self):
		return self.title



class MenuSelection(models.Model):
	title = models.CharField(null=False, max_length=100)
	details = models.CharField(max_length=600)
	menu = models.ForeignKey(Menu,
		related_name='menu_selection',
		on_delete=models.CASCADE)
	created_at = models.DateTimeField(auto_now_add=True)

	class Meta:
		ordering = ('title',)

	def __str__(self):
		return self.title



class MenuItem(models.Model):
	title = models.CharField(null=False, max_length=100)
	details = models.CharField(max_length=600)
	price = models.DecimalField(max_digits=10, decimal_places=2)
	menu_selection = models.ForeignKey(MenuSelection,
		on_delete=models.CASCADE)
	image = models.ImageField(upload_to='uploads/MenuItems/', null=True)
	created_at = models.DateTimeField(auto_now_add=True)

	class Meta:
		ordering = ('title',)

	def __str__(self):
		return self.title




class DrinkSelection(models.Model):
	name = models.CharField(null=False, max_length=100)
	details = models.CharField(max_length=600)
	created_at = models.DateTimeField(auto_now_add=True)

	class Meta:
		ordering = ('name',)

	def __str__(self):
		return self.name




class Drink(models.Model):
	name = models.CharField(null=False, max_length=100)
	details = models.CharField(max_length=600)
	price = models.DecimalField(max_digits=10, decimal_places=2)
	drink_selection = models.ForeignKey(DrinkSelection,
		on_delete=models.CASCADE)
	created_at = models.DateTimeField(auto_now_add=True)
	image = models.ImageField(upload_to='uploads/drinks/', null=True)

	class Meta:
		ordering = ('name',)

	def __str__(self):
		return self.name



class MealItem(models.Model):
	name = models.CharField(null=False, max_length=100)
	details = models.CharField(max_length=600, null=True)
	menu_item = models.ManyToManyField(MenuItem)
	quantity = models.PositiveSmallIntegerField(default=1)
	created_at = models.DateTimeField(auto_now_add=True)

	class Meta:
		ordering = ('name',)

	def __str__(self):
		return self.name



class Meal(models.Model):
	name = models.CharField(null=False, max_length=100)
	details = models.CharField(max_length=600)
	meal_items = models.ForeignKey(MealItem, null=True, on_delete=models.CASCADE)
	drinks = models.ManyToManyField(Drink)
	created_at = models.DateTimeField(auto_now_add=True)

	class Meta:
		ordering = ('name',)

	def __str__(self):
		return self.name




class Order(models.Model):
	created_by = models.OneToOneField(user_model, null=True, blank=True, on_delete=models.CASCADE)
	meal = models.ManyToManyField(Meal,
		related_name='orders')
	NEW = 0
	RECIEVED = 1
	PREPARING = 2
	COMPLETE = 3
	CANCELLED = 4
	ORDER_CHOICES = (
			(NEW, 'New'),
			(RECIEVED, 'Recieved'),
			(PREPARING, 'Preparing'),
			(COMPLETE, 'Complete'),
			(CANCELLED, 'Cancelled'),
		)
	table = models.CharField(max_length=10, null=True)
	status = models.SmallIntegerField(choices=ORDER_CHOICES, default=NEW)
	URGENT = 'U'
	NORMAL = 'N'
	DELIVERY_CHOICES = (
			(URGENT, 'Urgent'),
			(NORMAL, 'Normal'),
		)
	delivery_mode = models.CharField(max_length=2, choices=DELIVERY_CHOICES, default=NORMAL)
	is_paid = models.BooleanField(default=False)
	tip = models.DecimalField(max_digits=10, decimal_places=2, default=0.00)
	delivery_fee = models.DecimalField(max_digits=10, decimal_places=2, default=0.00)
	subtotal = models.DecimalField(max_digits=10, decimal_places=2, default=0.00)
	total = models.DecimalField(max_digits=10, decimal_places=2, default=0.00)
	created_at = models.DateTimeField(auto_now_add=True)
	updated_at = models.DateTimeField(auto_now=True)

	class Meta:
		ordering = ('id',)

	def __str__(self):
		return self.status



class Payment(models.Model):
 	amount= models.DecimalField(max_digits=10, decimal_places=2)
 	created_at = models.DateTimeField(auto_now_add=True)
 	UNPAID = 0
 	COMPLETE = 1
 	REFUNDED = 2
 	DECLINED = 3
 	SETTLING = 4
 	PAYMENT_CHOICES = (
			(UNPAID, 'Unpaid'),
			(COMPLETE, 'Complete'),
			(REFUNDED, 'Refunded'),
			(DECLINED, 'Declined'),
			(SETTLING, 'Settling'),
		)

 	status = models.SmallIntegerField(choices=PAYMENT_CHOICES, default=UNPAID)

 	class Meta:
 		ordering = ('created_at',)

 	def __str__(self):
 		return self.status




class Notification(models.Model):
	order = models.ForeignKey(Order,
		on_delete=models.SET_NULL, null=True)
	header = models.CharField(max_length=100)
	content = models.CharField(max_length=600)
	email = models.EmailField(max_length=50)
	created_at = models.DateTimeField(auto_now_add=True)

	class Meta:
		ordering = ('header',)

	def __str__(self):
		return self.header