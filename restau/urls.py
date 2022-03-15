from django.urls import re_path, path
from restau import views


urlpatterns = [
	re_path(r'^restaurants/$', views.RestaurantList.as_view()),
	re_path(r'^restaurants/(?P<pk>[0-9]+)$', views.RestaurantDetail.as_view()),
	re_path(r'^branches/$', views.BranchList.as_view()),
	re_path(r'^branch/(?P<pk>[0-9]+)$', views.BranchDetail.as_view()),
	re_path(r'^menus/$', views.MenuList.as_view()),
	re_path(r'^menu/(?P<pk>[0-9]+)$', views.MenuDetail.as_view()),
	re_path(r'^menu-selections/$', views.MenuSelectionList.as_view()),
	re_path(r'^menu-selection/(?P<pk>[0-9]+)$', views.MenuSelectionDetail.as_view()),
	re_path(r'^menu-items/$', views.MenuItemList.as_view()),
	re_path(r'^menu-item/(?P<pk>[0-9]+)$', views.MenuItemDetail.as_view()),
	re_path(r'^drinks/$', views.DrinkList.as_view()),
	re_path(r'^drink/(?P<pk>[0-9]+)$', views.DrinkDetail.as_view()),
	re_path(r'^drink-selections/$', views.DrinkSelectionList.as_view()),
	re_path(r'^drink-selection/(?P<pk>[0-9]+)$', views.DrinkSelectionDetail.as_view()),
	re_path(r'^meal-items/$', views.MealItemlList.as_view()),
	re_path(r'^meal-item/(?P<pk>[0-9]+)$', views.MealItemDetail.as_view()),
	re_path(r'^meals/$', views.MealList.as_view()),
	re_path(r'^meal/(?P<pk>[0-9]+)$', views.MealDetail.as_view()),
	re_path(r'^orders/$', views.OrderList.as_view()),
	re_path(r'^order/(?P<pk>[0-9]+)$', views.OrderDetail.as_view()),
	re_path(r'^payments/$', views.PaymentList.as_view()),

	re_path(r'^$', views.UserList.as_view()),
	re_path(r'^user/(?P<pk>[0-9]+)$', views.UserDetail.as_view()),

	re_path('api/v1/accounts/register/customer/', views.CustomerRegister.as_view()),
	re_path('api/v1/accounts/register/receptionist/', views.ReceptionistRegister.as_view()),
	re_path('api/v1/accounts/register/chef/', views.ChefRegister.as_view()),
	re_path('api/v1/accounts/register/waiter/', views.WaiterRegister.as_view()),
	re_path('api/v1/accounts/register/manager/', views.ManagerRegister.as_view()),
	re_path('api/v1/accounts/register/owner/', views.OwnerRegister.as_view()),
]