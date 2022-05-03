from django.urls import re_path, path
from restau import views


urlpatterns = [

	re_path(r'^$', views.ApiRoot.as_view(), name = views.ApiRoot.name),

	re_path(r'^restaurants/$', views.RestaurantList.as_view(), name = views.RestaurantList.name),
	re_path(r'^restaurants/(?P<pk>[0-9]+)$', views.RestaurantDetail.as_view(), name = views.RestaurantDetail.name),
	re_path(r'^branches/$', views.BranchList.as_view(), name = views.BranchList.name),
	re_path(r'^branch/(?P<pk>[0-9]+)$', views.BranchDetail.as_view(), name = views.BranchDetail.name),
	re_path(r'^menus/$', views.MenuList.as_view(), name = views.MenuList.name),
	re_path(r'^menu/(?P<pk>[0-9]+)$', views.MenuDetail.as_view(), name = views.MenuDetail.name),
	re_path(r'^menu-selections/$', views.MenuSelectionList.as_view(), name = views.MenuSelectionList.name),
	re_path(r'^menu-selection/(?P<pk>[0-9]+)$', views.MenuSelectionDetail.as_view(), name = views.MenuSelectionDetail.name),
	re_path(r'^menu-items/$', views.MenuItemList.as_view(), name = views.MenuItemList.name),
	re_path(r'^menu-item/(?P<pk>[0-9]+)$', views.MenuItemDetail.as_view(), name = views.MenuItemDetail.name),
	re_path(r'^drinks/$', views.DrinkList.as_view(), name = views.DrinkList.name),
	re_path(r'^drink/(?P<pk>[0-9]+)$', views.DrinkDetail.as_view(), name = views.DrinkDetail.name),
	re_path(r'^drink-selections/$', views.DrinkSelectionList.as_view(), name = views.DrinkSelectionList.name),
	re_path(r'^drink-selection/(?P<pk>[0-9]+)$', views.DrinkSelectionDetail.as_view(), name = views.DrinkSelectionDetail.name),
	re_path(r'^meal-items/$', views.MealItemlList.as_view(), name = views.MealItemlList.name),
	re_path(r'^meal-item/(?P<pk>[0-9]+)$', views.MealItemDetail.as_view(), name = views.MealItemDetail.name),
	re_path(r'^meals/$', views.MealList.as_view(), name = views.MealList.name),
	re_path(r'^meal/(?P<pk>[0-9]+)$', views.MealDetail.as_view(), name = views.MealDetail.name),
	re_path(r'^orders/$', views.OrderList.as_view(), name = views.OrderList.name),
	re_path(r'^order/(?P<pk>[0-9]+)$', views.OrderDetail.as_view(), name = views.OrderDetail.name),
	re_path(r'^payments/$', views.PaymentList.as_view(), name = views.PaymentList.name),

	re_path(r'^users/$', views.UserList.as_view(), name = views.UserList.name),
	re_path(r'^user/(?P<pk>[0-9]+)$', views.UserDetail.as_view(), name = views.UserDetail.name),

	re_path('api/v1/accounts/register/customer/', views.CustomerRegister.as_view()),
	re_path('api/v1/accounts/register/receptionist/', views.ReceptionistRegister.as_view()),
	re_path('api/v1/accounts/register/chef/', views.ChefRegister.as_view()),
	re_path('api/v1/accounts/register/waiter/', views.WaiterRegister.as_view()),
	re_path('api/v1/accounts/register/manager/', views.ManagerRegister.as_view()),
	re_path('api/v1/accounts/register/owner/', views.OwnerRegister.as_view()),
]