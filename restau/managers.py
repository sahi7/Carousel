from django.contrib.auth.base_user import BaseUserManager
from django.utils import timezone

class UserManager(BaseUserManager):
	"""
	Custom user model manager.
	"""
	def get_by_natural_key(self, username):
		return self.get(username=username)


	def create_superuser(self, username, email, password, **extra_fields):
		username = username

		if username is None:
			raise TypeError('Username is required')
		super_user = self.model(
				username=username, email=self.normalize_email(email), **extra_fields
			)
		super_user.is_active=True
		super_user.is_staff=True
		super_user.is_superuser=True
		super_user.date_created=timezone.now()
		super_user.set_password(password)
		super_user.save()
		return super_user




class CustomerManager(BaseUserManager):

	def create_customer(self, name, surname, email, gender, username, password=None, **extra_fields):
		if username is None:
			raise TypeError('Username is required.')
		customer = self.model(
				name=name, surname=surname, username=username,
				gender=gender, email=self.normalize_email(email),
			)
		customer.is_active = True
		customer.is_customer=True
		customer.set_password(password)
		customer.save()
		return customer



class ReceptionistManager(BaseUserManager):

	def create_receptionist(self, name, surname, email, username, password=None, **extra_fields):
		if username is None:
			raise TypeError('Username is required.')
		receptionist = self.model(
				name=name, surname=surname, username=username,
				email=self.normalize_email(email),
			)
		receptionist.is_active=True
		receptionist.is_receptionist=True
		receptionist.set_password(password)
		receptionist.save()
		return receptionist



class ChefManager(BaseUserManager):

	def create_chef(self, name, surname, email, username, password=None, **extra_fields):
		if username is None:
			raise TypeError('Username is required.')
		chef = self.model(
				name=name, surname=surname, username=username, address=address, phone_number=phone_number,
				email=self.normalize_email(email),
			)
		chef.is_active=True
		chef.is_chef=True
		chef.set_password(password)
		chef.save()
		return chef



class WaiterManager(BaseUserManager):

	def create_waiter(self, name, surname, email, username, password=None, **extra_fields):
		if username is None:
			raise TypeError('Username is required.')
		waiter = self.model(
				name=name, surname=surname, username=username,
				email=self.normalize_email(email),
			)
		waiter.is_active=True
		waiter.is_waiter=True
		waiter.set_password(password)
		waiter.save()
		return waiter



class RestManagerManager(BaseUserManager):

	def create_manager(self, name, surname, email, username, password=None, **extra_fields):
		if username is None:
			raise TypeError('Username is required.')
		manager = self.model(
				name=name, surname=surname, username=username,
				email=self.normalize_email(email),
			)
		manager.is_active=True
		manager.is_manager=True
		manager.set_password(password)
		manager.save()
		return manager



class OwnerManager(BaseUserManager):

	def create_owner(self, name, surname, email, username, password=None, **extra_fields):
		if username is None:
			raise TypeError('Username is required.')
		owner = self.model(
				name=name, surname=surname, username=username,
				email=self.normalize_email(email),
			)
		owner.is_active=True
		owner.is_owner=True
		owner.set_password(password)
		owner.save()
		return owner