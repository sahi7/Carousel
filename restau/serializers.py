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
	owner = serializers.ReadOnlyField(source='owner.username')
	manager = serializers.SlugRelatedField(queryset=User.objects.filter(is_manager=True),
		slug_field='username')

	class Meta:
		model = Restaurant
		fields = ('id', 'name', 'address', 'owner', 'manager', 'created_at', 'opened_on')


class BranchSerializer(serializers.HyperlinkedModelSerializer):
	restaurant = serializers.ReadOnlyField(source='restaurant.name')

	class Meta:
		model = Branch
		fields = ('url', 'id', 'name', 'address', 'details', 'restaurant', 'created_at', 'opened_on',)




class MenuSerializer(serializers.HyperlinkedModelSerializer):

	class Meta:
		model = Menu
		fields = ('url', 'id', 'title', 'details', 'created_at',)
 


class MenuSelectionSerializer(serializers.HyperlinkedModelSerializer):
	menu = serializers.SlugRelatedField(queryset=Menu.objects.all(),
		slug_field='nom',)

	class Meta:
		model = MenuSelection
		fields = ('url', 'id', 'title', 'details', 'menu', 'created_at',)



class MenuItemSerializer(serializers.HyperlinkedModelSerializer):

	class Meta:
		model = MenuItem
		fields = ('url', 'id', 'title', 'price', 'menu_selection',)




class DrinkSelectionSerializer(serializers.HyperlinkedModelSerializer):

	class Meta:
		model = DrinkSelection
		fields = ('url', 'name', 'created_at',)



class DrinkSerializer(serializers.HyperlinkedModelSerializer):

	class Meta:
		model = Drink
		fields = ('url', 'price', 'drink_selection', 'created_at',)



class MealItemSerializer(serializers.HyperlinkedModelSerializer):

	class Meta:
		model = MealItem
		fields = ('url', 'menu_item', 'quantity',)



class MealSerializer(serializers.HyperlinkedModelSerializer):
	meal_items = serializers.SlugRelatedField(
        many=True,
        read_only=True,
        slug_field='name'
     )

	class Meta:
		model = Meal
		fields = ('url', 'id', 'meal_items', 'drinks')



class OrderSerializer(serializers.HyperlinkedModelSerializer):
	status = serializers.ChoiceField(choices=Order.ORDER_CHOICES)

	class Meta:
		model = Order
		fields = ('url', 'id', 'created_by', 'meal', 'status')



class PaymentSerializer(serializers.HyperlinkedModelSerializer):
	status = serializers.ChoiceField(choices=Payment.PAYMENT_CHOICES)

	class Meta:
		model = Payment
		fields = ('url', 'id', 'status',)










	