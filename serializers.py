from rest_registration.api.serializers import DefaultRegisterUserSerializer
from rest_registration.utils.users import get_user_public_field_names

from django.contrib.auth import get_user_model
from rest_framework import serializers
from .models import Restaurant, Branch, MenuItem, MenuSelection, Menu, DrinkSelection, Drink, MealItem, Meal
from .models import Order, Payment, Notification, User, Manager, Customer, Receptionist, Chef, Waiter, Owner


class MetaObj:
    pass


class UserSerializer(serializers.ModelSerializer):

	class Meta:
		model = get_user_model()
		fields = ('id', 'name', 'surname', 'username', 'email', 'address', 'phone_number', 'created_at', 'user_status')


class CustomerRegisterSerializer(DefaultRegisterUserSerializer):

    def create(self, validated_data):
        data = validated_data.copy()
        if self.has_password_confirm_field():
            del data['password_confirm']
        return Customer.objects.create_customer(**data)


class ReceptionistRegisterSerializer(DefaultRegisterUserSerializer):

    def create(self, validated_data):
        data = validated_data.copy()
        if self.has_password_confirm_field():
            del data['password_confirm']
        return Receptionist.objects.create_receptionist(**data)


class ChefRegisterSerializer(DefaultRegisterUserSerializer):

    def create(self, validated_data):
        data = validated_data.copy()
        if self.has_password_confirm_field():
            del data['password_confirm']
        return Chef.objects.create_chef(**data)


class WaiterRegisterSerializer(DefaultRegisterUserSerializer):

    def create(self, validated_data):
        data = validated_data.copy()
        if self.has_password_confirm_field():
            del data['password_confirm']
        return Waiter.objects.create_waiter(**data)


class ManagerRegisterSerializer(DefaultRegisterUserSerializer):

    def create(self, validated_data):
        data = validated_data.copy()
        if self.has_password_confirm_field():
            del data['password_confirm']
        return Manager.objects.create_manager(**data)


class OwnerRegisterSerializer(DefaultRegisterUserSerializer):

    def create(self, validated_data):
        data = validated_data.copy()
        if self.has_password_confirm_field():
            del data['password_confirm']
        return Owner.objects.create_owner(**data)


class RestaurantSerializer(serializers.HyperlinkedModelSerializer):
	owner = serializers.SlugRelatedField(queryset=Owner.objects.all(),
		slug_field='name',)

	class Meta:
		model = Restaurant
		fields = ('url', 'owner', 'id', 'name', 'address', 'created_at', 'opened_on')


class BranchSerializer(serializers.HyperlinkedModelSerializer):

	class Meta:
		model = Branch
		fields = ('url', 'id', 'name', 'address', 'details', 'restaurant', 'created_at', 'opened_on',)

	def to_representation(self, instance):
		response = super().to_representation(instance)
		response['restaurant'] = RestaurantSerializer(instance.restaurant, context={'request': None}).data
		return response



class MenuSerializer(serializers.HyperlinkedModelSerializer):

	class Meta:
		model = Menu
		fields = ('url', 'id', 'title', 'details', 'branch', 'created_at',)

 


class MenuSelectionSerializer(serializers.HyperlinkedModelSerializer):
	menu = serializers.SlugRelatedField(queryset=Menu.objects.all(),
		slug_field='title',)

	class Meta:
		model = MenuSelection
		fields = ('url', 'id', 'title', 'details', 'menu', 'created_at',)



class MenuItemSerializer(serializers.HyperlinkedModelSerializer):

	class Meta:
		model = MenuItem
		fields = ('url', 'id', 'title', 'details', 'price', 'menu_selection',)




class DrinkSelectionSerializer(serializers.HyperlinkedModelSerializer):

	class Meta:
		model = DrinkSelection
		fields = ('url', 'id', 'name', 'branch', 'created_at',)



class DrinkSerializer(serializers.HyperlinkedModelSerializer):

	class Meta:
		model = Drink
		fields = ('url', 'id', 'name', 'price', 'drink_selection', 'created_at',)



class MealItemSerializer(serializers.HyperlinkedModelSerializer):

	class Meta:
		model = MealItem
		fields = ('url', 'id', 'menu_item', 'quantity',)



class MealSerializer(serializers.HyperlinkedModelSerializer):


	class Meta:
		model = Meal
		fields = ('url', 'id', 'meal_items', 'drinks')



class OrderSerializer(serializers.HyperlinkedModelSerializer):
	status = serializers.ChoiceField(choices=Order.ORDER_CHOICES)
	created_by = serializers.SlugRelatedField(queryset=User.objects.all(),
		slug_field='name',)

	class Meta:
		model = Order
		fields = '__all__'



class PaymentSerializer(serializers.HyperlinkedModelSerializer):
	status = serializers.ChoiceField(choices=Payment.PAYMENT_CHOICES)

	class Meta:
		model = Payment
		fields = ('url', 'id', 'status',)










	