package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Кастомные ошибки
var (
	ErrInsufficientFunds   = errors.New("недостаточно средств на счете")
	ErrInvalidAmount       = errors.New("некорректная сумма (отрицательная или нулевая)")
	ErrAccountNotFound     = errors.New("счет не найден")
	ErrSameAccountTransfer = errors.New("попытка перевода на тот же счёт")
)

// Тип транзакции
type TransactionType string

const (
	Deposit  TransactionType = "DEPOSIT"
	Withdraw TransactionType = "WITHDRAW"
	Transfer TransactionType = "TRANSFER"
)

// Транзакция
type Transaction struct {
	Type      TransactionType
	Amount    float64
	Timestamp time.Time
	Message   string
}

// Счет
type Account struct {
	ID           string
	OwnerName    string
	Balance      float64
	Transactions []Transaction
}

// AccountService - основной интерфейс для работы со счетом
type AccountService interface {
	Deposit(amount float64) error
	Withdraw(amount float64) error
	Transfer(to *Account, amount float64) error
	GetBalance() float64
	GetStatement() string
}

// Storage - интерфейс для работы с хранилищем данных
type Storage interface {
	SaveAccount(account *Account) error
	LoadAccount(accountID string) (*Account, error)
	GetAllAccounts() ([]*Account, error)
}

// Реализация AccountService
type BankAccountService struct {
	account *Account
	storage Storage
}

func NewBankAccountService(account *Account, storage Storage) *BankAccountService {
	return &BankAccountService{
		account: account,
		storage: storage,
	}
}

func (s *BankAccountService) Deposit(amount float64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	s.account.Balance += amount
	s.account.Transactions = append(s.account.Transactions, Transaction{
		Type:      Deposit,
		Amount:    amount,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Пополнение счета на %.2f", amount),
	})

	return s.storage.SaveAccount(s.account)
}

func (s *BankAccountService) Withdraw(amount float64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	if s.account.Balance < amount {
		return ErrInsufficientFunds
	}

	s.account.Balance -= amount
	s.account.Transactions = append(s.account.Transactions, Transaction{
		Type:      Withdraw,
		Amount:    amount,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Снятие средств на %.2f", amount),
	})

	return s.storage.SaveAccount(s.account)
}

func (s *BankAccountService) Transfer(to *Account, amount float64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	if s.account.Balance < amount {
		return ErrInsufficientFunds
	}

	if s.account.ID == to.ID {
		return ErrSameAccountTransfer
	}

	// Снимаем средства с текущего счета
	s.account.Balance -= amount
	s.account.Transactions = append(s.account.Transactions, Transaction{
		Type:      Transfer,
		Amount:    amount,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Перевод счету %s на %.2f", to.ID, amount),
	})

	// Пополняем целевой счет
	to.Balance += amount
	to.Transactions = append(to.Transactions, Transaction{
		Type:      Transfer,
		Amount:    amount,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Перевод от счета %s на %.2f", s.account.ID, amount),
	})

	// Сохраняем оба счета
	if err := s.storage.SaveAccount(s.account); err != nil {
		return err
	}
	return s.storage.SaveAccount(to)
}

func (s *BankAccountService) GetBalance() float64 {
	return s.account.Balance
}

func (s *BankAccountService) GetStatement() string {
	if len(s.account.Transactions) == 0 {
		return "История транзакций пуста"
	}

	var sb strings.Builder
	sb.WriteString("Выписка по счету:\n")
	sb.WriteString("Дата/Время | Тип | Сумма | Описание\n")
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	for _, tx := range s.account.Transactions {
		sb.WriteString(fmt.Sprintf("%s | %s | %.2f | %s\n",
			tx.Timestamp.Format("2006-01-02 15:04:05"),
			tx.Type,
			tx.Amount,
			tx.Message,
		))
	}

	return sb.String()
}

// InMemoryStorage - реализация хранилища в памяти
type InMemoryStorage struct {
	accounts map[string]*Account
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		accounts: make(map[string]*Account),
	}
}

func (s *InMemoryStorage) SaveAccount(account *Account) error {
	s.accounts[account.ID] = account
	return nil
}

func (s *InMemoryStorage) LoadAccount(accountID string) (*Account, error) {
	account, exists := s.accounts[accountID]
	if !exists {
		return nil, ErrAccountNotFound
	}
	return account, nil
}

func (s *InMemoryStorage) GetAllAccounts() ([]*Account, error) {
	accounts := make([]*Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		accounts = append(accounts, account)
	}
	return accounts, nil
}

// Генератор ID счетов
var accountCounter = 1

func generateAccountID() string {
	id := fmt.Sprintf("ACC%06d", accountCounter)
	accountCounter++
	return id
}

// Вспомогательные функции
func createAccount(storage Storage, scanner *bufio.Scanner) (*Account, error) {
	fmt.Print("Введите имя владельца счета: ")
	scanner.Scan()
	ownerName := strings.TrimSpace(scanner.Text())

	if ownerName == "" {
		return nil, errors.New("имя владельца не может быть пустым")
	}

	account := &Account{
		ID:        generateAccountID(),
		OwnerName: ownerName,
		Balance:   0,
	}

	if err := storage.SaveAccount(account); err != nil {
		return nil, err
	}

	return account, nil
}

func printAccounts(storage Storage) {
	accounts, err := storage.GetAllAccounts()
	if err != nil {
		fmt.Printf("Ошибка при получении списка счетов: %v\n", err)
		return
	}

	if len(accounts) == 0 {
		fmt.Println("Счета не найдены")
		return
	}

	fmt.Println("\nСписок счетов:")
	for _, account := range accounts {
		fmt.Printf("ID: %s, Владелец: %s, Баланс: %.2f\n",
			account.ID, account.OwnerName, account.Balance)
	}
}

func main() {
	storage := NewInMemoryStorage()
	scanner := bufio.NewScanner(os.Stdin)
	var currentAccountService *BankAccountService

	fmt.Println("=== Банковская система ===")

	for {
		if currentAccountService == nil {
			fmt.Println("\n1. Создать счет")
			fmt.Println("2. Выбрать счет")
			fmt.Println("3. Показать все счета")
			fmt.Println("4. Выйти")
		} else {
			fmt.Printf("\nТекущий счет: %s (%s)\n",
				currentAccountService.account.ID,
				currentAccountService.account.OwnerName)
			fmt.Println("1. Пополнить счет")
			fmt.Println("2. Снять средства")
			fmt.Println("3. Перевести другому счету")
			fmt.Println("4. Просмотреть баланс")
			fmt.Println("5. Получить выписку")
			fmt.Println("6. Сменить счет")
			fmt.Println("7. Выйти")
		}

		fmt.Print("Выберите действие: ")
		scanner.Scan()
		choice := strings.TrimSpace(scanner.Text())

		if currentAccountService == nil {
			switch choice {
			case "1":
				account, err := createAccount(storage, scanner)
				if err != nil {
					fmt.Printf("Ошибка при создании счета: %v\n", err)
				} else {
					fmt.Printf("Счет создан успешно! ID: %s\n", account.ID)
				}

			case "2":
				printAccounts(storage)
				fmt.Print("Введите ID счета: ")
				scanner.Scan()
				accountID := strings.TrimSpace(scanner.Text())

				account, err := storage.LoadAccount(accountID)
				if err != nil {
					fmt.Printf("Ошибка: %v\n", err)
				} else {
					currentAccountService = NewBankAccountService(account, storage)
					fmt.Printf("Счет %s выбран успешно!\n", accountID)
				}

			case "3":
				printAccounts(storage)

			case "4":
				fmt.Println("До свидания!")
				return

			default:
				fmt.Println("Неверный выбор. Попробуйте снова.")
			}
		} else {
			switch choice {
			case "1": // Пополнение
				fmt.Print("Введите сумму для пополнения: ")
				scanner.Scan()
				amount, err := strconv.ParseFloat(strings.TrimSpace(scanner.Text()), 64)
				if err != nil {
					fmt.Println("Ошибка: некорректная сумма")
					continue
				}

				if err := currentAccountService.Deposit(amount); err != nil {
					fmt.Printf("Ошибка при пополнении: %v\n", err)
				} else {
					fmt.Println("Счет успешно пополнен!")
				}

			case "2": // Снятие
				fmt.Print("Введите сумму для снятия: ")
				scanner.Scan()
				amount, err := strconv.ParseFloat(strings.TrimSpace(scanner.Text()), 64)
				if err != nil {
					fmt.Println("Ошибка: некорректная сумма")
					continue
				}

				if err := currentAccountService.Withdraw(amount); err != nil {
					fmt.Printf("Ошибка при снятии: %v\n", err)
				} else {
					fmt.Println("Средства успешно сняты!")
				}

			case "3": // Перевод
				printAccounts(storage)
				fmt.Print("Введите ID счета получателя: ")
				scanner.Scan()
				toAccountID := strings.TrimSpace(scanner.Text())

				fmt.Print("Введите сумму для перевода: ")
				scanner.Scan()
				amount, err := strconv.ParseFloat(strings.TrimSpace(scanner.Text()), 64)
				if err != nil {
					fmt.Println("Ошибка: некорректная сумма")
					continue
				}

				toAccount, err := storage.LoadAccount(toAccountID)
				if err != nil {
					fmt.Printf("Ошибка: %v\n", err)
					continue
				}

				if err := currentAccountService.Transfer(toAccount, amount); err != nil {
					fmt.Printf("Ошибка при переводе: %v\n", err)
				} else {
					fmt.Println("Перевод выполнен успешно!")
				}

			case "4": // Баланс
				balance := currentAccountService.GetBalance()
				fmt.Printf("Текущий баланс: %.2f\n", balance)

			case "5": // Выписка
				statement := currentAccountService.GetStatement()
				fmt.Println(statement)

			case "6": // Сменить счет
				currentAccountService = nil
				fmt.Println("Счет сброшен")

			case "7": // Выйти
				fmt.Println("До свидания!")
				return

			default:
				fmt.Println("Неверный выбор. Попробуйте снова.")
			}
		}
	}
}
