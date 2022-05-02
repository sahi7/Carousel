from restau import serializers
from rest_registration.api.views import register
from django.shortcuts import render
from rest_framework import generics
from rest_framework.generics import CreateAPIView
from rest_framework.reverse import reverse
from rest_framework.response import Response
<<<<<<< HEAD
from rest_framework.permissions import IsAuthenticated, AllowAny
=======
from rest_framework.permissions import IsAuthenticated
>>>>>>> 3a517a2335cc9f2d5939afd22149f4749019f035
from .models import Restaurant, Branch, MenuItem, MenuSelection, Menu, DrinkSelection, Drink, MealItem, Meal
from .models import Order, Payment, Notification, User, Customer


class CustomerRegister(CreateAPIView):
	queryset = Customer.objects.all()
	serializer_class = serializers.CustomerRegisterSerializer
	name = 'customer-register'

class WaiterRegister(CreateAPIView):
	serializer_class = serializers.WaiterRegisterSerializer
	name = 'waiter-register'

class ChefRegister(CreateAPIView):
	serializer_class = serializers.ChefRegisterSerializer
	name = 'chef-register'

class ReceptionistRegister(CreateAPIView):
	serializer_class = serializers.ReceptionistRegisterSerializer
	name = 'receptionist-register'

class ManagerRegister(CreateAPIView):
	serializer_class = serializers.ManagerRegisterSerializer
	name = 'manager-register'

class OwnerRegister(CreateAPIView):
	serializer_class = serializers.OwnerRegisterSerializer
	name = 'owner-register'

class UserList(generics.ListCreateAPIView):
	queryset = User.objects.all()
	serializer_class = serializers.UserSerializer
	name = 'user-list'

class UserDetail(generics.RetrieveUpdateDestroyAPIView):
	queryset = User.objects.all()
	serializer_class = serializers.UserSerializer
	name = 'user-detail'


class RestaurantList(generics.ListCreateAPIView):
	permission_classes = [IsAuthenticated]
	queryset = Restaurant.objects.all()
	serializer_class = serializers.RestaurantSerializer
	name = 'restaurant-list'

<<<<<<< HEAD
	def post(self, request, *args, **kwargs):
		#Getting current user(owner) from request.data
		request.data['owner'] = request.user.name
		return super(RestaurantList, self).post(request, *args, **kwargs)

=======
>>>>>>> 3a517a2335cc9f2d5939afd22149f4749019f035
class RestaurantDetail(generics.RetrieveUpdateDestroyAPIView):
	queryset = Restaurant.objects.all()
	serializer_class = serializers.RestaurantSerializer
	name = 'restaurant-detail'


class BranchList(generics.ListCreateAPIView):
<<<<<<< HEAD
	permission_classes = [AllowAny]
=======
>>>>>>> 3a517a2335cc9f2d5939afd22149f4749019f035
	queryset = Branch.objects.all()
	serializer_class = serializers.BranchSerializer
	name = 'branch-list'

<<<<<<< HEAD

=======
>>>>>>> 3a517a2335cc9f2d5939afd22149f4749019f035
class BranchDetail(generics.RetrieveUpdateDestroyAPIView):
	queryset = Branch.objects.all()
	serializer_class = serializers.BranchSerializer
	name = 'branch-detail'


class MenuList(generics.ListCreateAPIView):
<<<<<<< HEAD
	permission_classes = [AllowAny]
=======
>>>>>>> 3a517a2335cc9f2d5939afd22149f4749019f035
	queryset = Menu.objects.all()
	serializer_class = serializers.MenuSerializer
	name = 'menu-list'

class MenuDetail(generics.RetrieveUpdateDestroyAPIView):
	queryset = Menu.objects.all()
	serializer_class = serializers.MenuSerializer
	name = 'menu-detail'


class MenuSelectionList(generics.ListCreateAPIView):
<<<<<<< HEAD
	permission_classes = [AllowAny]
=======
>>>>>>> 3a517a2335cc9f2d5939afd22149f4749019f035
	queryset = MenuSelection.objects.all()
	serializer_class = serializers.MenuSelectionSerializer
	name = 'menu_selection-list'

class MenuSelectionDetail(generics.RetrieveUpdateDestroyAPIView):
	queryset = MenuSelection.objects.all()
	serializer_class = serializers.MenuSelectionSerializer
	name = 'menu_selection-detail'


class MenuItemList(generics.ListCreateAPIView):
	queryset = MenuItem.objects.all()
	serializer_class = serializers.MenuItemSerializer
	name = 'menu_item-list'

class MenuItemDetail(generics.RetrieveUpdateDestroyAPIView):
	queryset = MenuItem.objects.all()
	serializer_class = serializers.MenuItemSerializer
	name = 'menu_item-detail'


class DrinkSelectionList(generics.ListCreateAPIView):
	queryset = DrinkSelection.objects.all()
	serializer_class = serializers.DrinkSelectionSerializer
	name = 'drink_selection-list'

class DrinkSelectionDetail(generics.RetrieveUpdateDestroyAPIView):
	queryset = DrinkSelection.objects.all()
	serializer_class = serializers.DrinkSelectionSerializer
	name = 'drink_selection-detail'


class DrinkList(generics.ListCreateAPIView):
	queryset = Drink.objects.all()
	serializer_class = serializers.DrinkSerializer
	name = 'drinks-list'

class DrinkDetail(generics.RetrieveUpdateDestroyAPIView):
	queryset = Drink.objects.all()
	serializer_class = serializers.DrinkSerializer
	name = 'drink-detail'


class MealItemlList(generics.ListCreateAPIView):
	queryset = MealItem.objects.all()
	serializer_class = serializers.MealItemSerializer
	name = 'meal_item-list'

class MealItemDetail(generics.RetrieveUpdateDestroyAPIView):
	queryset = MealItem.objects.all()
	serializer_class = serializers.MealItemSerializer
	name = 'meal_item-detail'


class MealList(generics.ListCreateAPIView):
	queryset = Meal.objects.all()
	serializer_class = serializers.MealSerializer
	name = 'meal-list'

class MealDetail(generics.RetrieveUpdateDestroyAPIView):
	queryset = Meal.objects.all()
	serializer_class = serializers.MealSerializer
	name = 'meal-detail'


class OrderList(generics.ListCreateAPIView):
	queryset = Order.objects.all()
	serializer_class = serializers.OrderSerializer
	name = 'order-list'

class OrderDetail(generics.RetrieveUpdateDestroyAPIView):
	queryset = Order.objects.all()
	serializer_class = serializers.OrderSerializer
	name = 'order-detail'


class PaymentList(generics.ListCreateAPIView):
	queryset = Payment.objects.all()
	serializer_class = serializers.PaymentSerializer
	name = 'payment-list'



<<<<<<< HEAD

class ApiRoot(generics.GenericAPIView):
	permission_classes = [AllowAny]
=======
"""
class ApiRoot(generics.GenericAPIView):
>>>>>>> 3a517a2335cc9f2d5939afd22149f4749019f035
	name = 'api-root'

	def get(self, request, *args, **kwargs):
		return Response({
			'Restaurants': reverse(RestaurantList.name, request=request),
			'Branches': reverse(BranchList.name, request=request),
			'Menus': reverse(MenuList.name, request=request),
			'Menu Selections': reverse(MenuSelectionList.name, request=request),
			'Menu Items': reverse(MenuItemList.name, request=request),
			'Drink Selections': reverse(DrinkSelectionList.name, request=request),
<<<<<<< HEAD
			'Drinks': reverse(DrinkList.name, request=request),
=======
			'Drink': reverse(DrinkList.name, request=request),
>>>>>>> 3a517a2335cc9f2d5939afd22149f4749019f035
			'Meal Items': reverse(MealItemlList.name, request=request),
			'Meals': reverse(MealList.name, request=request),
			'Orders': reverse(OrderList.name, request=request),
			'Payments': reverse(PaymentList.name, request=request),
			})

<<<<<<< HEAD

"""
class RestaurantBranchView():

	def perform_create(self, serializer):
	restaurant_pk = self.kwargs['restaurant_pk']
	restaurant = get_object_or_404(Restaurant.objects.all(), pk=restaurant_pk)

	serializer.save(author=self.request.user, restaurant=restaurant)

=======
>>>>>>> 3a517a2335cc9f2d5939afd22149f4749019f035
"""